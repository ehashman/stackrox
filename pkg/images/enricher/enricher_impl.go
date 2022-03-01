package enricher

import (
	"context"
	"fmt"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/integrationhealth"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/time/rate"
)

const (
	// The number of consecutive errors for a scanner or registry that cause its health status to be UNHEALTHY
	consecutiveErrorThreshold = 3
)

var (
	_ ImageEnricher = (*enricherImpl)(nil)
)

type enricherImpl struct {
	cvesSuppressor   cveSuppressor
	cvesSuppressorV2 cveSuppressor
	integrations     integration.Set

	errorsPerRegistry  map[registryTypes.ImageRegistry]int32
	registryErrorsLock sync.RWMutex
	errorsPerScanner   map[scannerTypes.ImageScannerWithDataSource]int32
	scannerErrorsLock  sync.RWMutex

	integrationHealthReporter integrationhealth.Reporter ``

	metadataLimiter *rate.Limiter
	metadataCache   expiringcache.Cache

	imageGetter imageGetter

	asyncRateLimiter *rate.Limiter

	metrics metrics
}

// EnrichWithVulnerabilities enriches the given image with vulnerabilities.
func (e *enricherImpl) EnrichWithVulnerabilities(image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (EnrichmentResult, error) {
	scanners := e.integrations.ScannerSet()
	if scanners.IsEmpty() {
		return EnrichmentResult{
			ScanResult: ScanNotDone,
		}, errors.New("no image scanners are integrated")
	}

	for _, imageScanner := range scanners.GetAll() {
		scanner := imageScanner.GetScanner()
		if vulnScanner, ok := scanner.(scannerTypes.ImageVulnerabilityGetter); ok {
			// Clairify is the only supported ImageVulnerabilityGetter at this time.
			if scanner.Type() != clairify.TypeString {
				log.Errorf("unexpected image vulnerability getter: %s [%s]", scanner.Name(), scanner.Type())
				continue
			}

			res, err := e.enrichWithVulnerabilities(scanner.Name(), imageScanner.DataSource(), vulnScanner, image, components, notes)
			if err != nil {
				return EnrichmentResult{
					ScanResult: ScanNotDone,
				}, errors.Wrapf(err, "retrieving image vulnerabilities from %s [%s]", scanner.Name(), scanner.Type())
			}

			return EnrichmentResult{
				ImageUpdated: res != ScanNotDone,
				ScanResult:   res,
			}, nil
		}
	}

	return EnrichmentResult{
		ScanResult: ScanNotDone,
	}, errors.New("no image vulnerability retrievers are integrated")
}

func (e *enricherImpl) enrichWithVulnerabilities(scannerName string, dataSource *storage.DataSource, scanner scannerTypes.ImageVulnerabilityGetter,
	image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (ScanResult, error) {
	scanStartTime := time.Now()
	scan, err := scanner.GetVulnerabilities(image, components, notes)
	e.metrics.SetImageVulnerabilityRetrievalTime(scanStartTime, scannerName, err)
	if err != nil || scan == nil {
		return ScanNotDone, err
	}

	enrichImage(image, scan, dataSource)

	return ScanSucceeded, nil
}

// EnrichImage enriches an image with the integration set present.
func (e *enricherImpl) EnrichImage(ctx EnrichmentContext, image *storage.Image) (EnrichmentResult, error) {
	errorList := errorhelpers.NewErrorList("image enrichment")

	imageNoteSet := make(map[storage.Image_Note]struct{}, len(image.Notes))
	for _, note := range image.Notes {
		imageNoteSet[note] = struct{}{}
	}

	updatedMetadata, err := e.enrichWithMetadata(ctx, image)
	errorList.AddError(err)
	if image.GetMetadata() == nil {
		imageNoteSet[storage.Image_MISSING_METADATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_METADATA)
	}

	scanResult, err := e.enrichWithScan(ctx, image)
	errorList.AddError(err)
	if scanResult == ScanNotDone && image.GetScan() == nil {
		imageNoteSet[storage.Image_MISSING_SCAN_DATA] = struct{}{}
	} else {
		delete(imageNoteSet, storage.Image_MISSING_SCAN_DATA)
	}

	image.Notes = image.Notes[:0]
	for note := range imageNoteSet {
		image.Notes = append(image.Notes, note)
	}

	e.cvesSuppressor.EnrichImageWithSuppressedCVEs(image)
	if features.VulnRiskManagement.Enabled() {
		e.cvesSuppressorV2.EnrichImageWithSuppressedCVEs(image)
	}

	return EnrichmentResult{
		ImageUpdated: updatedMetadata || (scanResult != ScanNotDone),
		ScanResult:   scanResult,
	}, errorList.ToError()
}

func (e *enricherImpl) enrichWithMetadata(ctx EnrichmentContext, image *storage.Image) (bool, error) {
	// Attempt to short-circuit before checking registries.
	metadataOutOfDate := metadataIsOutOfDate(image.GetMetadata())
	if !metadataOutOfDate {
		return false, nil
	}

	if ctx.FetchOpt != ForceRefetch {
		// The metadata in the cache is always up-to-date with respect to the current metadataVersion
		if metadataValue := e.metadataCache.Get(getRef(image)); metadataValue != nil {
			e.metrics.IncrementMetadataCacheHit()
			image.Metadata = metadataValue.(*storage.ImageMetadata).Clone()
			return true, nil
		}
		e.metrics.IncrementMetadataCacheMiss()
	}
	if ctx.FetchOpt == NoExternalMetadata {
		return false, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error getting metadata for image: %s", image.GetName().GetFullName()))

	if image.GetName().GetRegistry() == "" {
		errorList.AddError(errors.New("no registry is indicated for image"))
		return false, errorList.ToError()
	}

	registries := e.integrations.RegistrySet()
	if !ctx.Internal && registries.IsEmpty() {
		errorList.AddError(errox.Newf(errox.NotFound, "no image registries are integrated: please add an image integration for %s", image.GetName().GetRegistry()))
		return false, errorList.ToError()
	}

	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	for _, registry := range registries.GetAll() {
		updated, err := e.enrichImageWithRegistry(image, registry)
		if err != nil {
			var currentRegistryErrors int32
			concurrency.WithLock(&e.registryErrorsLock, func() {
				currentRegistryErrors = e.errorsPerRegistry[registry] + 1
				e.errorsPerRegistry[registry] = currentRegistryErrors
			})

			if currentRegistryErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            registry.DataSource().Id,
					Name:          registry.DataSource().Name,
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: timestamp.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)
			continue
		}
		if updated {
			var currentRegistryErrors int32
			concurrency.WithRLock(&e.registryErrorsLock, func() {
				currentRegistryErrors = e.errorsPerRegistry[registry]
			})
			if currentRegistryErrors > 0 {
				concurrency.WithLock(&e.registryErrorsLock, func() {
					if e.errorsPerRegistry[registry] != currentRegistryErrors {
						return
					}
					e.errorsPerRegistry[registry] = 0
				})
			}
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            registry.DataSource().Id,
				Name:          registry.DataSource().Name,
				Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
				Status:        storage.IntegrationHealth_HEALTHY,
				LastTimestamp: timestamp.TimestampNow(),
				ErrorMessage:  "",
			})
			return true, nil
		}
	}

	if !ctx.Internal && len(errorList.ErrorStrings()) == 0 {
		errorList.AddError(errors.Errorf("no matching image registries found: please add an image integration for %s", image.GetName().GetRegistry()))
	}

	return false, errorList.ToError()
}

func getRef(image *storage.Image) string {
	if image.GetId() != "" {
		return image.GetId()
	}
	return image.GetName().GetFullName()
}

func (e *enricherImpl) enrichImageWithRegistry(image *storage.Image, registry registryTypes.ImageRegistry) (bool, error) {
	if !registry.Match(image.GetName()) {
		return false, nil
	}

	// Wait until limiter allows entrance
	_ = e.metadataLimiter.Wait(context.Background())
	metadata, err := registry.Metadata(image)
	if err != nil {
		return false, errors.Wrapf(err, "error getting metadata from registry: %q", registry.Name())
	}
	metadata.DataSource = registry.DataSource()
	metadata.Version = metadataVersion
	image.Metadata = metadata

	cachedMetadata := metadata.Clone()
	e.metadataCache.Add(getRef(image), cachedMetadata)
	if image.GetId() == "" {
		if digest := image.Metadata.GetV2().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, cachedMetadata)
		}
		if digest := image.Metadata.GetV1().GetDigest(); digest != "" {
			e.metadataCache.Add(digest, cachedMetadata)
		}
	}
	return true, nil
}

func (e *enricherImpl) fetchFromDatabase(img *storage.Image, option FetchOption) bool {
	if option == ForceRefetch || option == ForceRefetchScansOnly {
		return false
	}
	// See if the image exists in the DB with a scan, if it does, then use that instead of fetching
	id := utils.GetImageID(img)
	if id == "" {
		return false
	}
	existingImage, exists, err := e.imageGetter(sac.WithAllAccess(context.Background()), id)
	if err != nil {
		log.Errorf("error fetching image %q: %v", id, err)
		return false
	}
	if exists && existingImage.GetScan() != nil {
		img.Scan = existingImage.GetScan()
		return true
	}
	return false
}

func (e *enricherImpl) enrichWithScan(ctx EnrichmentContext, image *storage.Image) (ScanResult, error) {
	// Attempt to short-circuit before checking scanners.
	if ctx.FetchOnlyIfScanEmpty() && image.GetScan() != nil {
		return ScanNotDone, nil
	}
	if e.fetchFromDatabase(image, ctx.FetchOpt) {
		return ScanSucceeded, nil
	}

	if ctx.FetchOpt == NoExternalMetadata {
		return ScanNotDone, nil
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning image: %s", image.GetName().GetFullName()))
	scanners := e.integrations.ScannerSet()
	if !ctx.Internal && scanners.IsEmpty() {
		errorList.AddError(errors.New("no image scanners are integrated"))
		return ScanNotDone, errorList.ToError()
	}

	for _, scanner := range scanners.GetAll() {
		result, err := e.enrichImageWithScanner(image, scanner)
		if err != nil {
			var currentScannerErrors int32
			concurrency.WithLock(&e.scannerErrorsLock, func() {
				currentScannerErrors = e.errorsPerScanner[scanner] + 1
				e.errorsPerScanner[scanner] = currentScannerErrors
			})
			if currentScannerErrors >= consecutiveErrorThreshold { // update health
				e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
					Id:            scanner.DataSource().Id,
					Name:          scanner.DataSource().Name,
					Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
					Status:        storage.IntegrationHealth_UNHEALTHY,
					LastTimestamp: timestamp.TimestampNow(),
					ErrorMessage:  err.Error(),
				})
			}
			errorList.AddError(err)
			continue
		}
		if result != ScanNotDone {
			var currentScannerErrors int32
			concurrency.WithRLock(&e.scannerErrorsLock, func() {
				currentScannerErrors = e.errorsPerScanner[scanner]
			})
			if currentScannerErrors > 0 {
				concurrency.WithLock(&e.scannerErrorsLock, func() {
					if e.errorsPerScanner[scanner] != currentScannerErrors {
						return
					}
					e.errorsPerScanner[scanner] = 0
				})
			}
			e.integrationHealthReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            scanner.DataSource().Id,
				Name:          scanner.DataSource().Name,
				Type:          storage.IntegrationHealth_IMAGE_INTEGRATION,
				Status:        storage.IntegrationHealth_HEALTHY,
				LastTimestamp: timestamp.TimestampNow(),
				ErrorMessage:  "",
			})
			return result, nil
		}
	}
	return ScanNotDone, errorList.ToError()
}

func normalizeVulnerabilities(scan *storage.ImageScan) {
	for _, c := range scan.GetComponents() {
		for _, v := range c.GetVulns() {
			v.Severity = cvss.VulnToSeverity(v)
		}
	}
}

func (e *enricherImpl) enrichImageWithScanner(image *storage.Image, imageScanner scannerTypes.ImageScannerWithDataSource) (ScanResult, error) {
	scanner := imageScanner.GetScanner()

	if !scanner.Match(image.GetName()) {
		return ScanNotDone, nil
	}

	sema := scanner.MaxConcurrentScanSemaphore()
	_ = sema.Acquire(context.Background(), 1)
	defer sema.Release(1)

	scanStartTime := time.Now()
	scan, err := scanner.GetScan(image)
	e.metrics.SetScanDurationTime(scanStartTime, scanner.Name(), err)
	if err != nil {
		return ScanNotDone, errors.Wrapf(err, "Error scanning %q with scanner %q", image.GetName().GetFullName(), scanner.Name())
	}
	if scan == nil {
		return ScanNotDone, nil
	}

	enrichImage(image, scan, imageScanner.DataSource())

	return ScanSucceeded, nil
}

func enrichImage(image *storage.Image, scan *storage.ImageScan, dataSource *storage.DataSource) {
	// Normalize the vulnerabilities.
	normalizeVulnerabilities(scan)

	scan.DataSource = dataSource

	// Assume:
	//  scan != nil
	//  no error scanning.
	image.Scan = scan
	FillScanStats(image)
}

// FillScanStats fills in the higher level stats from the scan data.
func FillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}

		var fixedByProvided bool
		var imageTopCVSS float32
		vulns := make(map[string]bool)
		for _, c := range i.GetScan().GetComponents() {
			var componentTopCVSS float32
			var hasVulns bool
			for _, v := range c.GetVulns() {
				hasVulns = true
				if _, ok := vulns[v.GetCve()]; !ok {
					vulns[v.GetCve()] = false
				}

				if v.GetCvss() > componentTopCVSS {
					componentTopCVSS = v.GetCvss()
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCve()] = true
				}
			}

			if hasVulns {
				c.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: componentTopCVSS,
				}
			}

			if componentTopCVSS > imageTopCVSS {
				imageTopCVSS = componentTopCVSS
			}
		}

		i.SetCves = &storage.Image_Cves{
			Cves: int32(len(vulns)),
		}

		if len(vulns) > 0 {
			i.SetTopCvss = &storage.Image_TopCvss{
				TopCvss: imageTopCVSS,
			}
		}

		if int32(len(vulns)) == 0 || fixedByProvided {
			var numFixableVulns int32
			for _, fixable := range vulns {
				if fixable {
					numFixableVulns++
				}
			}
			i.SetFixable = &storage.Image_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
}
