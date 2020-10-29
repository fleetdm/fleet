import React, { Component } from 'react';

const baseClass = 'osquery-options';

class OsqueryOptionsPage extends Component {

    render () {
        return (
            <div className={`${baseClass} body-wrap`}>
                <h1>Osquery Options</h1>
            </div>
        );
    };
}

export default OsqueryOptionsPage;