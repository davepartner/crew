import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./styles.css";

// Resolve the theme before the first paint so there is no flash.
const saved = window.localStorage.getItem("crew-theme") ?? "system";
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
document.documentElement.dataset.theme = saved === "system" ? (prefersDark ? "dark" : "light") : saved;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
