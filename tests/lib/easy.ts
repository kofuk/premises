import { assertEquals } from "https://deno.land/std@0.83.0/testing/asserts.ts";

import api, { params, streamEvent } from "../lib/api.ts";
import codes from "../lib/codes.ts";

const waitServerLaunched = async (cookie: string) => {
  for (let i = 0; i < 18; i++) {
    console.log(".");
    await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

    const { pageCode } = await streamEvent("/api/streaming/events", cookie);
    if (pageCode === codes.PAGE.RUNNING) {
      break;
    } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
      throw new Error("Unexpected page");
    }
  }
  const { pageCode } = await streamEvent("/api/streaming/events", cookie);
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

export const launchNewWorld = async (
  cookie: string,
  worldName: string,
): Promise<void> => {
  await api(
    "POST /api/launch",
    cookie,
    params({
      "machine-type": "2g",
      "server-version": "1.20.1",
      "prefer-detect": "true",
      "world-source": "new-world",
      "world-name": worldName,
      "seed": "",
      "level-type": "default",
    }),
  );

  await waitServerLaunched(cookie);
};

export const launchExistingWorld = async (
  cookie: string,
  worldName: string,
): Promise<void> => {
  await api(
    "POST /api/launch",
    cookie,
    params({
      "machine-type": "2g",
      "server-version": "1.20.1",
      "prefer-detect": "true",
      "world-source": "backups",
      "world-name": worldName,
      "backup-generation": "@/latest",
    }),
  );

  await waitServerLaunched(cookie);
};

export const stopServer = async (cookie: string): Promise<void> => {
  await api("POST /api/stop", cookie);

  for (let i = 0; i < 18; i++) {
    console.log(".");
    await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

    const { pageCode } = await streamEvent("/api/streaming/events", cookie);
    if (pageCode === codes.PAGE.LAUNCH) {
      break;
    } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
      throw new Error("Unexpected page");
    }
  }
};
