import { writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const placeholder = fileURLToPath(
  new URL("../out/placeholder.txt", import.meta.url),
);

await writeFile(
  placeholder,
  "This placeholder keeps the go:embed pattern `web/out/*` (uiembed.go) satisfied\n" +
    "in clean checkouts and `go install` builds where the Next.js viewer has not\n" +
    "been exported. Run `make build` (or build under web/) to produce the real\n" +
    "dashboard assets; the embedded viewer serves them when present. Builds that\n" +
    "should skip the viewer entirely can use `-tags noembed`.\n",
  "utf8",
);
