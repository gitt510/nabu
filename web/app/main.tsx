import { createRoot } from "react-dom/client";

// CSS は package のもの（Mantine と書体）だけ。自前の CSS は持たない
import "@mantine/core/styles.css";
import "@fontsource-variable/noto-sans-jp";
import "@fontsource-variable/noto-sans-mono";

import { App } from "./App";
import { Provider } from "./components/ui";

// vite の dev server（just dev）で開いた画面は、tab の題で deploy 版と見分ける
if (import.meta.env.DEV) document.title += " (dev)";

const root = document.getElementById("root");
if (!root) throw new Error("#root not found in index.html");
createRoot(root).render(
  <Provider>
    <App />
  </Provider>,
);
