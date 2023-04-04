import { useEffect } from "react";
import { FitAddon } from "xterm-addon-fit";
import { Terminal } from "xterm";
import "xterm/css/xterm.css";

let mode: string;
if (!process.env.NODE_ENV || process.env.NODE_ENV === "development") {
  mode = "dev";
} else {
  mode = "prod";
}

export function WebLog({ exec }: { exec: string }) {
  useEffect(() => {
    if (exec != "") {
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
      return () => {
        ws.close();
        terminal.dispose();
        window.removeEventListener("resize", resize);
      };
    }
  }, [exec]);

  return (
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
  );
}
