import { useState } from "react";
import { FileUploader } from "react-drag-drop-files";

export function Service({ name, exec }: { name: string; exec: string }) {
  const [file, setFile] = useState<File | null>(null);
  const handleChange = (file: any) => {
    console.log(file);
    setFile(file);
  };
  return (
    <div>
      <label>{name}</label>
      <hr />
      <FileUploader handleChange={handleChange} name="file" />
      <button
        disabled={!file}
        onClick={() => {
          const formData = new FormData();
          formData.append("file", file!, "binary");
          fetch(`/api/service/${exec}/upload`, {
            method: "POST",
            body: formData,
          })
            .then((res) => res.json())
            .then((res: any) => alert(res.error || "成功"));
        }}
      >
        更新
      </button>
      <button>开始</button>
      <button>停止</button>
    </div>
  );
}
