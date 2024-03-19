import locale from "../../controlpanel/frontend/i18n/en.json" with {
  type: "json",
};

const error = (code: number): string => {
  const message = locale[`error.code_${code}`];
  if (message == null) {
    throw Error(`No message found for error code: ${code}`);
  }
  return message as string;
};

const PAGE = {
  LAUNCH: 1,
  LOADING: 2,
  RUNNING: 3,
  MANUAL_SETUP: 4,
};

export default { error, PAGE };
