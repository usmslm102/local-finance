/**
 * Note: When using the Node.JS APIs, the config file
 * doesn't apply. Instead, pass options directly to the APIs.
 *
 * All configuration options: https://remotion.dev/docs/config
 */

import { existsSync } from "node:fs";
import { Config } from "@remotion/cli/config";

Config.setRspack(false);
Config.setVideoImageFormat("jpeg");
Config.setOverwriteOutput(true);
Config.setDefaultPremountInSeconds(2);

// The downloaded Headless Shell 157 times out on this macOS host.
// Prefer an existing local Chrome; other hosts retain Remotion's default.
const chromePath =
  process.env.REMOTION_BROWSER_EXECUTABLE ||
  (process.platform === "darwin"
    ? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    : "");
if (chromePath && existsSync(chromePath))
  Config.setBrowserExecutable(chromePath);
