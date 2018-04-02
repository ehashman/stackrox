import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { actions as benchmarkActions } from 'reducers/benchmarks';
import dateFns from 'date-fns';
import { ClipLoader } from 'react-spinners';
import { sortNumber } from 'sorters/sorters';

import Table from 'Components/Table';
import Select from 'Components/Select';
import BenchmarksSidePanel from 'Containers/Compliance/BenchmarksSidePanel';
import HostResultModal from 'Containers/Compliance/HostResultModal';

class BenchmarksPage extends Component {
    static propTypes = {
        benchmarkScanResults: PropTypes.arrayOf(PropTypes.object).isRequired,
        lastScannedTime: PropTypes.string.isRequired,
        benchmarkName: PropTypes.string.isRequired,
        benchmarkId: PropTypes.string.isRequired,
        clusterId: PropTypes.string.isRequired,
        startPollBenchmarkScanResults: PropTypes.func.isRequired,
        stopPollBenchmarkScanResults: PropTypes.func.isRequired,
        selectBenchmarkScheduleDay: PropTypes.func.isRequired,
        selectBenchmarkScheduleHour: PropTypes.func.isRequired,
        selectBenchmarkScanResult: PropTypes.func.isRequired,
        selectBenchmarkHostResult: PropTypes.func.isRequired,
        fetchBenchmarkSchedule: PropTypes.func.isRequired,
        schedule: PropTypes.shape({
            day: PropTypes.string,
            hour: PropTypes.string
        }).isRequired,
        triggerBenchmarkScan: PropTypes.func.isRequired,
        selectedBenchmarkScanResult: PropTypes.shape({
            definition: PropTypes.shape({ name: PropTypes.string }),
            hostResults: PropTypes.arrayOf(PropTypes.object)
        }),
        selectedBenchmarkHostResult: PropTypes.shape({
            host: PropTypes.string,
            notes: PropTypes.arrayOf(PropTypes.string)
        })
    };

    static defaultProps = {
        selectedBenchmarkScanResult: null,
        selectedBenchmarkHostResult: null
    };

    constructor(props) {
        super(props);

        this.state = {
            scanning: false
        };
    }

    componentDidMount() {
        this.props.startPollBenchmarkScanResults({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
        this.props.fetchBenchmarkSchedule({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.lastScannedTime !== this.props.lastScannedTime) {
            // if new benchmark results are loaded then stop the button scanning if it is scanning
            this.setState({ scanning: false });
        }
    }

    componentWillUnmount() {
        this.props.stopPollBenchmarkScanResults();
    }

    onTriggerScan = () => {
        this.setState({ scanning: true });
        this.props.triggerBenchmarkScan({
            benchmarkId: this.props.benchmarkId,
            clusterId: this.props.clusterId
        });
    };

    onRowClick = benchmarkScanResult => {
        this.props.selectBenchmarkScanResult(benchmarkScanResult);
    };

    onCloseSidePanel = () => {
        this.props.selectBenchmarkScanResult(null);
    };

    onHostResultClick = benchmarkHostResult => {
        this.props.selectBenchmarkHostResult(benchmarkHostResult);
    };

    onBenchmarkHostResultModalClose = () => {
        this.props.selectBenchmarkHostResult(null);
    };

    onScheduleDayChange = value => {
        this.props.selectBenchmarkScheduleDay(
            this.props.benchmarkId,
            this.props.benchmarkName,
            value
        );
    };

    onScheduleHourChange = value => {
        this.props.selectBenchmarkScheduleHour(
            this.props.benchmarkId,
            this.props.benchmarkName,
            value,
            this.props.clusterId
        );
    };

    renderScanOptions = () => {
        const category = {
            options: [
                { label: 'None', value: null },
                { label: 'Monday', value: 'Monday' },
                { label: 'Tuesday', value: 'Tuesday' },
                { label: 'Wednesday', value: 'Wednesday' },
                { label: 'Thursday', value: 'Thursday' },
                { label: 'Friday', value: 'Friday' },
                { label: 'Saturday', value: 'Saturday' },
                { label: 'Sunday', value: 'Sunday' }
            ]
        };
        return (
            <Select
                className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded"
                value={this.props.schedule.day}
                placeholder="No scheduled scanning"
                options={category.options}
                onChange={this.onScheduleDayChange}
            />
        );
    };

    renderScanTimes = () => {
        const category = {
            options: [
                { label: '00:00 AM', value: '00:00 AM' },
                { label: '01:00 AM', value: '01:00 AM' },
                { label: '02:00 AM', value: '02:00 AM' },
                { label: '03:00 AM', value: '03:00 AM' },
                { label: '04:00 AM', value: '04:00 AM' },
                { label: '05:00 AM', value: '05:00 AM' },
                { label: '06:00 AM', value: '06:00 AM' },
                { label: '07:00 AM', value: '07:00 AM' },
                { label: '08:00 AM', value: '08:00 AM' },
                { label: '09:00 AM', value: '09:00 AM' },
                { label: '10:00 AM', value: '10:00 AM' },
                { label: '11:00 AM', value: '11:00 AM' },
                { label: '12:00 PM', value: '12:00 PM' },
                { label: '01:00 PM', value: '01:00 PM' },
                { label: '02:00 PM', value: '02:00 PM' },
                { label: '03:00 PM', value: '03:00 PM' },
                { label: '04:00 PM', value: '04:00 PM' },
                { label: '05:00 PM', value: '05:00 PM' },
                { label: '06:00 PM', value: '06:00 PM' },
                { label: '07:00 PM', value: '07:00 PM' },
                { label: '08:00 PM', value: '08:00 PM' },
                { label: '09:00 PM', value: '09:00 PM' },
                { label: '10:00 PM', value: '10:00 PM' },
                { label: '11:00 PM', value: '11:00 PM' }
            ]
        };
        return (
            <Select
                className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded"
                value={this.props.schedule.hour}
                placeholder="None"
                options={category.options}
                onChange={this.onScheduleHourChange}
            />
        );
    };

    renderScanButton = () => {
        const buttonScanning = (
            <button className="p-3 ml-5 h-10 w-24 rounded-sm bg-success-500 text-white hover:bg-success-600 uppercase text-center">
                <ClipLoader color="white" loading={this.state.scanning} size={20} />
            </button>
        );
        const scanButton = (
            <button
                className="p-3 ml-5 h-10 w-24 rounded-sm bg-success-500 text-white hover:bg-success-600 uppercase"
                onClick={this.onTriggerScan}
            >
                Scan now
            </button>
        );
        return this.state.scanning ? buttonScanning : scanButton;
    };

    renderTable = () => {
        const table = {
            columns: [
                { key: 'definition.name', label: 'Name' },
                { key: 'definition.description', label: 'Description' },
                {
                    key: 'aggregatedResults.PASS',
                    label: 'Pass',
                    default: 0,
                    align: 'right',
                    sortMethod: sortNumber('aggregatedResults.PASS')
                },
                {
                    key: 'aggregatedResults.INFO',
                    label: 'Info',
                    default: 0,
                    align: 'right',
                    sortMethod: sortNumber('aggregatedResults.INFO')
                },
                {
                    key: 'aggregatedResults.WARN',
                    label: 'Warn',
                    default: 0,
                    align: 'right',
                    sortMethod: sortNumber('aggregatedResults.WARN')
                },
                {
                    key: 'aggregatedResults.NOTE',
                    label: 'Note',
                    default: 0,
                    align: 'right',
                    sortMethod: sortNumber('aggregatedResults.NOTE')
                }
            ],
            rows: this.props.benchmarkScanResults
        };
        return <Table columns={table.columns} rows={table.rows} onRowClick={this.onRowClick} />;
    };

    renderModal() {
        if (!this.props.selectedBenchmarkHostResult) return '';
        return (
            <HostResultModal
                benchmarkHostResult={this.props.selectedBenchmarkHostResult}
                onClose={this.onBenchmarkHostResultModalClose}
            />
        );
    }

    renderBenchmarksSidePanel() {
        if (!this.props.selectedBenchmarkScanResult) return '';
        return (
            <BenchmarksSidePanel
                header={this.props.selectedBenchmarkScanResult.definition.name}
                hostResults={this.props.selectedBenchmarkScanResult.hostResults}
                onClose={this.onCloseSidePanel}
                onRowClick={this.onHostResultClick}
            />
        );
    }

    render() {
        return (
            <div className="flex flex-col h-full">
                <div className="flex w-full mb-3 px-3 items-center">
                    <span className="flex flex-1 text-xl font-500 text-primary-500 self-end">
                        Last Scanned: {this.props.lastScannedTime || 'Never'}
                    </span>
                    <div className="flex self-center justify-end pr-5 border-r border-primary-200">
                        <span className="mr-4">{this.renderScanOptions()}</span>
                        <span>{this.renderScanTimes()}</span>
                    </div>
                    {this.renderScanButton()}
                </div>
                <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                    <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                        {this.renderTable()}
                    </div>
                    {this.renderBenchmarksSidePanel()}
                    {this.renderModal()}
                </div>
            </div>
        );
    }
}

const getBenchmarkScanResults = createSelector([selectors.getLastScan], data => {
    const lastScan = data.response;
    if (!lastScan || !lastScan.metadata) return [];
    const { checks } = lastScan.data;
    return checks;
});

const getLastScannedTime = createSelector([selectors.getLastScan], data => {
    const lastScan = data.response;
    if (!lastScan || !lastScan.metadata) return '';
    const scanTime = dateFns.format(lastScan.metadata.time, 'MM/DD/YYYY h:mm:ss A');
    return scanTime || '';
});

const mapStateToProps = createStructuredSelector({
    benchmarkScanResults: getBenchmarkScanResults,
    lastScannedTime: getLastScannedTime,
    schedule: selectors.getBenchmarkSchedule,
    selectedBenchmarkScanResult: selectors.getSelectedBenchmarkScanResult,
    selectedBenchmarkHostResult: selectors.getSelectedBenchmarkHostResult
});

const mapDispatchToProps = dispatch => ({
    startPollBenchmarkScanResults: benchmark =>
        dispatch(benchmarkActions.pollBenchmarkScanResults.start(benchmark)),
    stopPollBenchmarkScanResults: () => dispatch(benchmarkActions.pollBenchmarkScanResults.stop()),
    selectBenchmarkScheduleDay: (benchmarkId, benchmarkName, value, clusterId) =>
        dispatch(
            benchmarkActions.selectBenchmarkScheduleDay(
                benchmarkId,
                benchmarkName,
                value,
                clusterId
            )
        ),
    selectBenchmarkScheduleHour: (benchmarkId, benchmarkName, value, clusterId) =>
        dispatch(
            benchmarkActions.selectBenchmarkScheduleHour(
                benchmarkId,
                benchmarkName,
                value,
                clusterId
            )
        ),
    fetchBenchmarkSchedule: benchmark =>
        dispatch(benchmarkActions.fetchBenchmarkSchedule.request(benchmark)),
    triggerBenchmarkScan: benchmark =>
        dispatch(benchmarkActions.triggerBenchmarkScan.request(benchmark)),
    selectBenchmarkScanResult: benchmarkScanResult =>
        dispatch(benchmarkActions.selectBenchmarkScanResult(benchmarkScanResult)),
    selectBenchmarkHostResult: benchmarkHostResult =>
        dispatch(benchmarkActions.selectBenchmarkHostResult(benchmarkHostResult))
});

export default connect(mapStateToProps, mapDispatchToProps)(BenchmarksPage);
