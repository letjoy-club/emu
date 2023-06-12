import { useEffect } from "react";
import { FitAddon } from "xterm-addon-fit";
import { Terminal } from "xterm";
import "xterm/css/xterm.css";
import { Button } from "@douyinfe/semi-ui";
import { IconRefresh } from "@douyinfe/semi-icons";
import { Subject } from "rxjs";

let mode: string;
if (!process.env.NODE_ENV || process.env.NODE_ENV === "development") {
  mode = "dev";
} else {
  mode = "prod";
}

const clear$ = new Subject<void>();

export function WebLog({ exec }: { exec: string }) {
  useEffect(() => {
    if (exec !== "") {
      const fitAddon = new FitAddon();
      const terminal = new Terminal();
      terminal.loadAddon(fitAddon);
      terminal.open(document.getElementById("terminal")!);
      terminal.clear();
      fitAddon.fit();
      terminal.writeln(`Connected To Server: [${exec}]`);

      function resize() {
        fitAddon.fit();
      }

      const ws = new WebSocket(
        `ws://` + (mode === "dev" ? "localhost:8080" : window.location.host) + `/api/service/${exec}/output`
      );
      ws.onmessage = (event) => {
        if (typeof event.data === "string") {
          terminal.write(event.data.replaceAll("\n", "\r\n"));
        }
      };

      window.addEventListener("resize", resize);

      const sub = clear$.subscribe(() => terminal.clear());
      return () => {
        ws.close();
        terminal.dispose();
        window.removeEventListener("resize", resize);
        sub.unsubscribe();
      };
    }
  }, [exec]);

  return (
    <>
      <div
        style={{
          borderLeft: "1px solid #b1b1b1",
          paddingLeft: 10,
          background: "black",
          flex: 1,
          height: "100%",
          boxSizing: "border-box",
        }}
      >
        <div id="terminal" style={{ height: "100vh" }}></div>
      </div>
      <Button
        style={{ position: "fixed", top: 15, right: 15, paddingLeft: 8, paddingRight: 8 }}
        type="secondary"
        size="small"
        theme="solid"
        onClick={() => clear$.next()}
        icon={<IconRefresh />}
      >
        清屏
      </Button>
    </>
  );
}
