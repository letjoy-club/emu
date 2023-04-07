import { Badge, Collapse, Layout, Tag } from "@douyinfe/semi-ui";
import { useEffect, useState } from "react";
import { Service } from "./service";
import { Typography } from "@douyinfe/semi-ui";
import { context } from "./context";
import { WebLog } from "./terminal";
import { TagColor } from "@douyinfe/semi-ui/lib/es/tag";
const { Title } = Typography;

const { Sider, Content } = Layout;

const Colors: TagColor[] = ["blue", "cyan", "green", "indigo", "orange", "pink", "purple"];
const tagColors: Map<string, TagColor> = new Map();

function App() {
  const [services, setServices] = useState<Service[]>([]);
  const [exec, setExec] = useState("");
  useEffect(() => {
    fetch("/api/service")
      .then((r) => r.json() as Promise<{ data: Service[] }>)
      .then((r) => {
        for (const service of r.data) {
          if (!tagColors.has(service.tag)) {
            const color = Colors.shift();
            console.log(service.tag, color);
            tagColors.set(service.tag, color!);
          }
        }
        console.log(tagColors);
        setServices(r.data);
      });
    const timer = setInterval(() => {
      fetch("/api/service")
        .then((r) => r.json() as Promise<{ data: Service[] }>)
        .then((r) => setServices(r.data));
    }, 2000);
    return () => clearInterval(timer);
  }, []);
  return (
    <context.Provider
      value={{
        update: () => {
          fetch("/api/service")
            .then((r) => r.json() as Promise<{ data: Service[] }>)
            .then((r) => setServices(r.data));
        },
        setExec,
      }}
    >
      <Layout>
        <Sider style={{ height: "100vh", overflowY: "auto" }}>
          <div style={{ width: 400 }}>
            <Title style={{ padding: 10 }}>服务管理</Title>
            <Collapse>
              {services.map((service, i) => (
                <Collapse.Panel
                  key={i}
                  header={
                    <>
                      <div>
                        {service.running ? (
                          <Badge dot style={{ backgroundColor: "var(--semi-color-success)" }} />
                        ) : (
                          <Badge dot type="danger" />
                        )}
                        {service.tag ? (
                          <Tag color={tagColors.get(service.tag)} style={{ marginLeft: 4 }}>
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
          </div>
        </Sider>

        <Content>
          <WebLog exec={exec} />
        </Content>
      </Layout>
    </context.Provider>
  );
}

export default App;
