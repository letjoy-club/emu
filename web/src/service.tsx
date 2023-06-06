import {
  Button,
  ButtonGroup,
  Dropdown,
  List,
  Modal,
  Popover,
  Space,
  Spin,
  Tag,
  Toast,
  Upload,
} from "@douyinfe/semi-ui";
import { useContext, useEffect, useRef, useState } from "react";
import { IconRefresh, IconStop, IconPlay, IconTerminal, IconFile, IconUpload } from "@douyinfe/semi-icons";
import { context } from "./context";
import { filesize } from "filesize";
import { Subject } from "rxjs";

export type Service = {
  name: string;
  exec: string;
  running: boolean;

  mem: number;
  cpu: number;
  pid: number;
  tag: string;
  fdNum: number;

  connections: string[];
  paths: string[];
};

type LogFile = {
  name: string;
  size: number;
};

export function Service({ service }: { service: Service }) {
  const [loading, setLoading] = useState(false);
  const ctx = useContext(context);
  const [logLoading, setLogLoading] = useState(false);
  const [logFiles, setLogFiles] = useState<LogFile[]>([]);

  const tags = service.connections.map((c, i) => (
    <Tag type="solid" key={i} color={c.startsWith(":::") ? "blue" : undefined}>
      端口：{c}
    </Tag>
  ));

  return (
    <div>
      {service.running ? (
        <Space wrap style={{ marginBottom: 8 }}>
          {tags}
          <Tag type="solid">
            <>PID: {service.pid}</>
          </Tag>
          <Tag type="solid">
            <>内存: {filesize(service.mem)}</>
          </Tag>
          <Tag type="solid">
            <>CPU: {service.cpu.toFixed(2)}%</>
          </Tag>
          <PathCard paths={service.paths}>
            <Tag color="teal" type="solid">
              <>句柄数: {service.fdNum}</>
            </Tag>
          </PathCard>
        </Space>
      ) : null}
      <div style={{ display: "flex" }}>
        <ButtonGroup>
          <Button icon={<IconUpload />} onClick={() => UploadModal$.next(service.exec)} />
          {service.running ? (
            <Button
              loading={loading}
              icon={<IconStop />}
              onClick={() => {
                setLoading(true);
                fetch(`/api/service/${service.exec}/stop`, { method: "POST" })
                  .then((r) => r.json())
                  .then((res) => {
                    setLoading(false);
                    if (res.error) {
                      Toast.error(res.error);
                    } else {
                      Toast.success("成功");
                    }
                  })
                  .finally(() => ctx.update());
              }}
              type="danger"
            />
          ) : (
            <Button
              loading={loading}
              icon={<IconPlay />}
              onClick={() => {
                setLoading(true);
                fetch(`/api/service/${service.exec}/start`, { method: "POST" })
                  .then((r) => r.json())
                  .then((res) => {
                    setLoading(false);
                    if (res.error) {
                      Toast.error(res.error);
                    } else {
                      Toast.success("成功");
                    }
                  })
                  .finally(() => ctx.update());
              }}
              type="secondary"
            />
          )}
        </ButtonGroup>
        <div style={{ flex: 1 }}></div>
        <ButtonGroup>
          <Dropdown
            trigger={"click"}
            position={"bottomLeft"}
            onVisibleChange={(visible) => {
              if (visible) {
                setLogLoading(true);
                fetch(`/api/service/${service.exec}/log`)
                  .then((r) => r.json())
                  .then((res) => {
                    if (res.error) {
                      Toast.error(res.error);
                    } else {
                      setLogFiles(res.data);
                    }
                    setLogLoading(false);
                  });
              } else {
              }
            }}
            render={
              logLoading ? (
                <div style={{ padding: 20, paddingTop: 30 }}>
                  <Spin size="large" />
                </div>
              ) : (
                <Dropdown.Menu>
                  {logFiles.map((f, i) => (
                    <Dropdown.Item
                      key={i}
                      type="secondary"
                      onClick={() => {
                        window.open("/api/service/" + service.exec + "/log/" + f.name, "_blank");
                      }}
                    >
                      <div>
                        <div>{f.name}</div>
                        <div style={{ fontSize: "0.8em" }}>
                          <>{filesize(f.size, { base: 2, standard: "jedec" })}</>
                        </div>
                      </div>
                    </Dropdown.Item>
                  ))}
                </Dropdown.Menu>
              )
            }
          >
            <Button icon={<IconFile />} />
          </Dropdown>
          <Button icon={<IconTerminal />} onClick={() => ctx.setExec(service.exec)}>
            输出
          </Button>
        </ButtonGroup>
      </div>
    </div>
  );
}

const UploadModal$ = new Subject<string>();

export function UploadModal() {
  const [show, setShow] = useState(false);
  const [loading, setLoading] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const ref = useRef<Upload>(null);
  const [exec, setExec] = useState("");

  useEffect(() => {
    const sub = UploadModal$.subscribe((exec) => {
      setExec(exec);
      setShow(true);
    });
    return () => sub.unsubscribe();
  }, []);
  return (
    <Modal
      title={"上传新的 " + exec}
      visible={show}
      onCancel={() => setShow(false)}
      footer={<Button onClick={() => setShow(false)}>关闭</Button>}
    >
      <Upload
        action={`/api/service/${exec}/upload`}
        style={{ margin: "10px 0" }}
        draggable={true}
        dragMainText={"点击上传文件或拖拽文件到这里"}
        name="file"
        ref={ref}
        fileName="binary"
        afterUpload={(result) => {
          setLoading(false);
          if (result.response.error) {
            Toast.error(result.response.error);
          } else {
            Toast.info("完成");
          }
          return {};
        }}
        beforeUpload={() => {
          setLoading(true);
          return true;
        }}
        onFileChange={(files) => {
          if (files.length > 0) {
            setFile(files[0]);
          } else {
            setFile(null);
          }
        }}
        limit={1}
        itemStyle={{ width: "100%" }}
      ></Upload>
      <Button
        icon={<IconRefresh />}
        loading={loading}
        disabled={!file}
        onClick={async () => {
          ref.current?.upload();
        }}
      >
        更新
      </Button>
    </Modal>
  );
}

function PathCard({ paths, children }: React.PropsWithChildren<{ paths: string[] }>) {
  return (
    <Popover
      showArrow
      content={<List size="small" dataSource={paths} renderItem={(item) => <List.Item>{item}</List.Item>} />}
    >
      {children}
    </Popover>
  );
}
