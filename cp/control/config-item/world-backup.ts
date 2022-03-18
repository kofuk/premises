export type GenerationInfo = {
    gen: string;
    id: string;
    timestamp: number;
};

export type WorldBackup = {
    worldName: string;
    generations: GenerationInfo[];
};
