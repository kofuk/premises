import * as React from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';

import '../i18n';
import {t} from 'i18next';

import ServerVersion from './config-item/server-version';
import WorldSource from './config-item/world-source';
import {WorldLocation} from './config-item/world-source';
import ChooseBackup from './config-item/choose-backup';
import WorldName from './config-item/world-name';
import ConfigureWorld from './config-item/configure-world';
import {LevelType} from './config-item/configure-world';

import {ServerConfig} from './server-config';

type Prop = {
    backToMenu: () => void;
    showError: (message: string) => void;
};

export default class ReconfigureMenu extends React.Component<Prop, ServerConfig> {
    state: ServerConfig = {
        machineType: '',
        serverVersion: '',
        worldSource: WorldLocation.Backups,
        worldName: '',
        backupGeneration: '',
        useCachedWorld: true,
        seed: '',
        levelType: LevelType.Default,
        currentStep: 0
    };
    stepCount: number = 0;

    handleStart = () => {
        const data = new URLSearchParams();
        data.append('server-version', this.state.serverVersion);
        data.append('world-source', this.state.worldSource);
        if (this.state.worldSource === WorldLocation.Backups) {
            data.append('world-name', this.state.worldName);
            data.append('backup-generation', this.state.backupGeneration);
            data.append('use-cache', this.state.useCachedWorld.toString());
        } else {
            data.append('world-name', this.state.worldName);
            data.append('seed', this.state.seed);
            data.append('level-type', this.state.levelType);
        }

        fetch('/control/api/reconfigure', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: data.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    this.props.backToMenu();
                } else {
                    this.props.showError(resp['message']);
                }
            });
    };

    setServerVersion = (serverVersion: string) => {
        this.setState({serverVersion: serverVersion});
    };

    setWorldSource = (worldSource: WorldLocation) => {
        this.setState({worldSource: worldSource});
        if (worldSource !== WorldLocation.Backups) {
            this.setState({worldName: ''});
        }
    };

    setWorldName = (worldName: string) => {
        this.setState({worldName: worldName});
    };

    setBackupGeneration = (generationId: string) => {
        this.setState({backupGeneration: generationId});
    };

    setUseCachedWorld = (useCachedWorld: boolean) => {
        this.setState({useCachedWorld: useCachedWorld});
    };

    setLevelType = (levelType: LevelType) => {
        this.setState({levelType: levelType});
    };

    setSeed = (seed: string) => {
        this.setState({seed: seed});
    };

    handleRequestFocus = (step: number) => {
        if (step < this.state.currentStep) {
            this.setState({currentStep: step});
        }
    };

    handleNextStep = () => {
        if (this.state.currentStep < this.stepCount) {
            this.setState({currentStep: this.state.currentStep + 1});
        }
    };

    render = () => {
        const configItems = [];
        {
            const stepIndex = configItems.length;
            configItems.push(
                <ServerVersion
                    key="serverVersion"
                    isFocused={this.state.currentStep === stepIndex}
                    nextStep={this.handleNextStep}
                    requestFocus={() => this.handleRequestFocus(stepIndex)}
                    stepNum={stepIndex + 1}
                    serverVersion={this.state.serverVersion}
                    setServerVersion={this.setServerVersion}
                />
            );
        }
        {
            const stepIndex = configItems.length;
            configItems.push(
                <WorldSource
                    key="worldSource"
                    isFocused={this.state.currentStep === stepIndex}
                    nextStep={this.handleNextStep}
                    requestFocus={() => this.handleRequestFocus(stepIndex)}
                    stepNum={stepIndex + 1}
                    worldSource={this.state.worldSource}
                    setWorldSource={this.setWorldSource}
                />
            );
        }

        if (this.state.worldSource === WorldLocation.Backups) {
            {
                const stepIndex = configItems.length;
                configItems.push(
                    <ChooseBackup
                        key="chooseBackup"
                        isFocused={this.state.currentStep === stepIndex}
                        nextStep={this.handleNextStep}
                        requestFocus={() => this.handleRequestFocus(stepIndex)}
                        stepNum={stepIndex + 1}
                        worldName={this.state.worldName}
                        backupGeneration={this.state.backupGeneration}
                        useCachedWorld={this.state.useCachedWorld}
                        setWorldName={this.setWorldName}
                        setBackupGeneration={this.setBackupGeneration}
                        setUseCachedWorld={this.setUseCachedWorld}
                    />
                );
            }
        } else {
            {
                const stepIndex = configItems.length;
                configItems.push(
                    <WorldName
                        key="worldName"
                        isFocused={this.state.currentStep === stepIndex}
                        nextStep={this.handleNextStep}
                        requestFocus={() => this.handleRequestFocus(stepIndex)}
                        stepNum={stepIndex + 1}
                        worldName={this.state.worldName}
                        setWorldName={this.setWorldName}
                    />
                );
            }
            {
                const stepIndex = configItems.length;
                configItems.push(
                    <ConfigureWorld
                        key="configureWorld"
                        isFocused={this.state.currentStep === stepIndex}
                        nextStep={this.handleNextStep}
                        requestFocus={() => this.handleRequestFocus(stepIndex)}
                        stepNum={stepIndex + 1}
                        levelType={this.state.levelType}
                        seed={this.state.seed}
                        setLevelType={this.setLevelType}
                        setSeed={this.setSeed}
                    />
                );
            }
        }

        this.stepCount = configItems.length;

        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    <IoIosArrowBack /> {t('back')}
                </button>
                <div className="m-2">
                    {configItems}
                    <div className="d-md-block mt-3 text-end">
                        <button
                            className="btn btn-primary bg-gradient"
                            type="button"
                            onClick={this.handleStart}
                            disabled={this.state.currentStep !== this.stepCount}
                        >
                            {t('relaunch_server')}
                        </button>
                    </div>
                </div>
            </div>
        );
    };
}
