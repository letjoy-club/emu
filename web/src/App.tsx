import { Badge, Collapse, Tag } from "@douyinfe/semi-ui";
import { useEffect, useState } from "react";
import { Service } from "./service";
import { Typography } from "@douyinfe/semi-ui";
import { context } from "./context";
const { Title } = Typography;

function App() {
  const [services, setServices] = useState<Service[]>([]);
  useEffect(() => {
    fetch("/api/service")
      .then((r) => r.json() as Promise<{ data: Service[] }>)
      .then((r) => setServices(r.data));
    const timer = setInterval(() => {
      fetch("/api/service")
        .then((r) => r.json() as Promise<{ data: Service[] }>)
        .then((r) => setServices(r.data));
    }, 2000);
    return () => clearInterval(timer);
  }, []);
  return (
    <div style={{ maxWidth: 400, margin: "5px auto 2px", border: "1px solid #d9d9d9" }}>
      <Title style={{ padding: 10 }}>服务管理</Title>
      <context.Provider
        value={{
          update: () => {
            fetch("/api/service")
              .then((r) => r.json() as Promise<{ data: Service[] }>)
              .then((r) => setServices(r.data));
          },
        }}
      ></context.Provider>
      <Collapse>
        {services.map((service, i) => (
          <Collapse.Panel
            header={
              <>
                <Tag>{i + 1}</Tag>
                <div>
                  {service.running ? (
                    <Badge dot style={{ backgroundColor: "var(--semi-color-success)" }} />
                  ) : (
                    <Badge dot type="danger" />
                  )}
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
  );
}

export default App;
