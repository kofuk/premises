import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { streamEvent } from "../lib/api.ts";
import codes from "../lib/codes.ts";

export const waitServerLaunched = async (cookie: string) => {
  for (let i = 0; i < 18; i++) {
    console.log(".");
    await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

    const { pageCode } = await streamEvent("/api/v1/streaming", cookie);
    if (pageCode === codes.PAGE.RUNNING) {
      break;
    } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
      throw new Error("Unexpected page");
    }
  }
  const { pageCode } = await streamEvent("/api/v1/streaming", cookie);
  assertEquals(pageCode, codes.PAGE.RUNNING);

  // It takes some time to initialize world info data.
  for (let i = 0; i < 10; i++) {
    try {
      await api("GET /api/worldinfo", cookie);
      break;
    } catch (_: unknown) {
      // Check error later
    }

    await new Promise((resolve) => setTimeout(resolve, 2 * 1000));
  }
};

export const initConfig = async (
  cookie: string,
  worldName: string,
): Promise<void> => {
  await api("GET /api/v1/config", cookie);
  await api("PUT /api/v1/config", cookie, {
    machineType: "2g",
    serverVersion: "1.20.1",
    guessServerVersion: true,
    worldSource: "new-world",
    worldName,
    levelType: "default",
  });
};

export const launchNewWorld = async (
  cookie: string,
  worldName: string,
): Promise<void> => {
  await initConfig(cookie, worldName);
  await api("POST /api/v1/launch", cookie);

  await waitServerLaunched(cookie);
};

export const launchExistingWorld = async (
  cookie: string,
  worldName: string,
): Promise<void> => {
  await initConfig(cookie, worldName);
  await api("PUT /api/v1/config", cookie, {
    worldSource: "backups",
    worldName,
    backupGen: "@/latest",
  });
  await api("POST /api/v1/launch", cookie);

  await waitServerLaunched(cookie);
};

export const stopServer = async (cookie: string): Promise<void> => {
  await api("POST /api/v1/stop", cookie);

  for (let i = 0; i < 18; i++) {
    console.log(".");
    await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

    const { pageCode } = await streamEvent("/api/v1/streaming", cookie);
    if (pageCode === codes.PAGE.LAUNCH) {
      break;
    } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
      throw new Error("Unexpected page");
    }
  }
};
