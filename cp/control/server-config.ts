import {WorldLocation} from './config-item/world-source';
import {LevelType} from './config-item/configure-world';

export type ServerConfig = {
    machineType: string,
    serverVersion: string,
    worldSource: WorldLocation,
    worldName: string,
    backupGeneration: string,
    useCachedWorld: boolean,
    seed: string,
    levelType: LevelType,
    currentStep: number
}
