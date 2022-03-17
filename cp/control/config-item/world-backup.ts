export type GenerationInfo = {
    gen: string,
    id: string
};

export type WorldBackup = {
    worldName: string,
    generations: GenerationInfo[]
};
