import { Badge, Button, Collapse, Tag } from "@douyinfe/semi-ui";
import { useEffect, useState } from "react";
import { ConfigSettingModal, IService, Service, UploadModal } from "./service";
import { Typography } from "@douyinfe/semi-ui";
import { context } from "./context";
import { WebLog } from "./terminal";
import { TagColor } from "@douyinfe/semi-ui/lib/es/tag";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
const { Title } = Typography;

const Colors: TagColor[] = [
  "blue",
  "cyan",
  "green",
  "indigo",
  "orange",
  "pink",
  "purple",
];
const tagColors: Map<string, TagColor> = new Map();

function App() {
  const [services, setServices] = useState<IService[]>([]);
  const [exec, setExec] = useState("");
  useEffect(() => {
    fetch("/api/config")
      .then(
        (r) =>
          r.json() as Promise<{
            data: { name: string; "metaVars": Record<string, string> };
          }>
      )
      .then((r) => {
        document.title = r.data.name;
      });
    fetch("/api/service")
      .then((r) => r.json() as Promise<{ data: IService[] }>)
      .then((r) => {
        for (const service of r.data) {
          if (!tagColors.has(service.tag)) {
            const color = Colors.shift();
            tagColors.set(service.tag, color!);
          }
        }
        setServices(r.data);
      });
    const timer = setInterval(() => {
      fetch("/api/service")
        .then((r) => r.json() as Promise<{ data: IService[] }>)
        .then((r) => setServices(r.data));
    }, 2000);
    return () => clearInterval(timer);
  }, []);
  return (
    <context.Provider
      value={{
        update: () => {
          fetch("/api/service")
            .then((r) => r.json() as Promise<{ data: IService[] }>)
            .then((r) => setServices(r.data));
        },
        setExec,
      }}
    >
      <PanelGroup direction="horizontal">
        <Panel
          defaultSize={20}
          minSize={20}
          style={{ maxHeight: "100vh", overflowY: "auto" }}
        >
          <Title style={{ padding: 10 }}>服务管理</Title>
          <Collapse>
            {services.map((service, i) => (
              <Collapse.Panel
                key={i}
                header={
                  <>
                    <div>
                      {service.running ? (
                        <Badge
                          dot
                          style={{
                            backgroundColor: "var(--semi-color-success)",
                          }}
                        />
                      ) : (
                        <Badge dot type="danger" />
                      )}
                      {service.tag ? (
                        <Tag
                          color={tagColors.get(service.tag)}
                          style={{ marginLeft: 4 }}
                        >
                          {service.tag}
                        </Tag>
                      ) : null}
                      {" " + service.name}
                    </div>
                  </>
                }
                itemKey={i.toString()}
              >
                <Service service={service} key={i} />
              </Collapse.Panel>
            ))}
          </Collapse>
          <UploadModal />
          <ConfigSettingModal />
        </Panel>
        <PanelResizeHandle className="ResizeHandle" />
        <Panel minSize={30}>
          <WebLog exec={exec} />
        </Panel>
      </PanelGroup>
    </context.Provider>
  );
}

export default App;
