import React, { useEffect, useState } from "react";
import { Service } from "./service";

function App() {
  const [services, setServices] = useState<{ name: string; exec: string }[]>([]);
  useEffect(() => {
    fetch("/api/service")
      .then((r) => r.json() as Promise<{ data: { name: string; exec: string }[] }>)
      .then((r) => setServices(r.data));
    console.log(services);
  }, []);
  return (
    <>
      {services.map((service, i) => (
        <Service name={service.name} exec={service.exec} key={i} />
      ))}
    </>
  );
}

export default App;
