import { createContext } from "react";

export const context = createContext({
  update: () => {},
  setExec: (name: string) => {},
});
