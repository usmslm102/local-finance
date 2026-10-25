import { Composition } from "remotion";
import { Film } from "./Film";
export const Root = () => (
  <Composition
    id="LocalFinance"
    component={Film}
    width={1080}
    height={1920}
    fps={30}
    durationInFrames={1800}
  />
);
