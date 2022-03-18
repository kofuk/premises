export type ItemProp = {
    isFocused: boolean;
    nextStep: () => void;
    requestFocus: () => void;
    stepNum: number;
};
