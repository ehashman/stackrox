import React from 'react';
import PropTypes from 'prop-types';

const activeTabHeaderClassName =
    'text-primary-100 bg-primary-500 border-2 border-primary-400 leading-none p-1 px-2 rounded-full';
const tabHeaderClassName =
    'border-2 border-primary-800 leading-none p-1 px-2 rounded-full text-primary-100 hover:bg-primary-800 hover:border-primary-700 ';

function NetworkEntityTabHeader({ title, isActive, onSelectTab }) {
    const className = isActive ? activeTabHeaderClassName : tabHeaderClassName;
    return (
        <li key={title} className="ml-2 first:ml-0">
            <button type="button" className={className} onClick={onSelectTab}>
                {title}
            </button>
        </li>
    );
}

NetworkEntityTabHeader.propTypes = {
    title: PropTypes.string.isRequired,
    isActive: PropTypes.bool.isRequired,
    onSelectTab: PropTypes.func.isRequired,
};

NetworkEntityTabHeader.defaultProps = {};

export default NetworkEntityTabHeader;
