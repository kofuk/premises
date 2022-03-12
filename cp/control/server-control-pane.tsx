import * as React from 'react';

export default class ServerControlPane extends React.Component<{}, {}> {
    render() {
        return (
            <div className="my-5 card mx-auto">
                <div className="card-body">
                    <form>
                        <div className="d-md-block mt-3 text-end">
                            <button className="btn btn-danger bg-gradient"
                                    type="button"
                                    onClick={(e: React.MouseEvent) => {e.preventDefault(); fetch('/control/api/stop', {method: 'post'});}}>
                                Stop
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        );
    };
};
