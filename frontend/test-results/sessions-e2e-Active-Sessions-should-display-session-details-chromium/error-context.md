# Page snapshot

```yaml
- generic [ref=e3]:
  - generic [ref=e4]: "[plugin:vite:esbuild] Transform failed with 1 error: /home/cmit/openpam/frontend/src/pages/analytics/index.ts:7:0: ERROR: Unexpected \"<<\""
  - generic [ref=e5]: /home/cmit/openpam/frontend/src/pages/analytics/index.ts:7:0
  - generic [ref=e6]: "Unexpected \"<<\" 5 | export { ReportDetailPage } from './ReportDetailPage'; 6 | export { ReportGeneratorPage } from './ReportGeneratorPage'; 7 | <<<<<<< HEAD | ^ 8 | export { ExceptionManagementPage } from './ExceptionManagementPage'; 9 | ======="
  - generic [ref=e7]: at failureErrorWithLog (/home/cmit/openpam/frontend/node_modules/esbuild/lib/main.js:1472:15) at /home/cmit/openpam/frontend/node_modules/esbuild/lib/main.js:755:50 at responseCallbacks.<computed> (/home/cmit/openpam/frontend/node_modules/esbuild/lib/main.js:622:9) at handleIncomingPacket (/home/cmit/openpam/frontend/node_modules/esbuild/lib/main.js:677:12) at Socket.readFromStdout (/home/cmit/openpam/frontend/node_modules/esbuild/lib/main.js:600:7) at Socket.emit (node:events:517:28) at addChunk (node:internal/streams/readable:368:12) at readableAddChunk (node:internal/streams/readable:341:9) at Readable.push (node:internal/streams/readable:278:10) at Pipe.onStreamRead (node:internal/stream_base_commons:190:23
  - generic [ref=e8]:
    - text: Click outside, press Esc key, or fix the code to dismiss.
    - text: You can also disable this overlay by setting
    - code [ref=e9]: server.hmr.overlay
    - text: to
    - code [ref=e10]: "false"
    - text: in
    - code [ref=e11]: vite.config.ts
    - text: .
```