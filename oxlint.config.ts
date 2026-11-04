// Lints the viewer in src/web/ so that it keeps no CSS of its own and touches Mantine and Tabler only inside
// src/web/components/ui/ (plus Mantine in app/Provider.tsx and app/Shell.tsx), and so that a ui part never picks its color by state. Each message says what to use
// instead, then why.
//
// no-restricted-imports carries several restrictions at once, and an override replaces its options whole, so
// every scope below restates the full list it needs.
import { defineConfig } from "oxlint";

const LIBRARY = {
  group: ["@mantine/**", "@tabler/**"],
  message: "Use components/ui instead: ui is the one door to Mantine and Tabler",
};
// Own CSS is imported by a relative or root path; package CSS (Mantine, fonts) by its package name.
const OWN_CSS = {
  regex: "^(\\.{1,2}/|/).*\\.css([?#].*)?$",
  message: "Use ui parts and the theme instead: nabu keeps no CSS of its own",
};
const PAGES = {
  regex: "^\\.\\./pages/",
  message: "Pass it in through props instead: components must not depend on pages",
};
const APP = {
  regex: "^(\\.\\./)+app/",
  message: "Pass it in through props instead: only main.tsx and App.tsx put the app together",
};
// app/Provider.tsx and app/Shell.tsx set up the whole app on Mantine, so they reach Mantine directly; icons
// still come through ui.
const TABLER = {
  group: ["@tabler/**"],
  message: "Use components/ui instead: ui is the one door to Tabler",
};
const OUTSIDE_UI = {
  regex: "^\\.\\./",
  message: "Pass it in through props instead: ui must not depend on anything outside ui",
};
const PASS_THROUGH = {
  regex: "^[^.]",
  message:
    "Wrap it in a ui file and export that instead: index.ts must not pass library parts through",
};
const imports = (...patterns: object[]) => ["error", { patterns }];

// Raw HTML tags stay allowed: without style or className (below) they carry no look. Only the two tags that
// bring CSS in on their own are forbidden.
const ELEMENTS = [
  { element: "style", message: "Use the theme instead: nabu keeps no CSS of its own" },
  {
    element: "link",
    message:
      "Import package CSS in components/ui/Provider.tsx instead: stylesheets load in one place",
  },
];

const LOOK = "Use the ui part's own props instead: the ui part decides the look";
const RENDER = "Use a ui part made for it instead: ui decides what is rendered";

export default defineConfig({
  plugins: ["react", "typescript", "unicorn", "oxc"],
  jsPlugins: ["./lint/nabu.js"],
  rules: {
    "no-restricted-imports": imports(LIBRARY, OWN_CSS),
    "react/forbid-elements": ["error", { forbid: ELEMENTS }],
    "react/forbid-dom-props": [
      "error",
      {
        forbid: ["style", "className"]
          .map((propName) => ({ propName, message: LOOK }))
          .concat({ propName: "dangerouslySetInnerHTML", message: RENDER }),
      },
    ],
    "react/forbid-component-props": [
      "error",
      {
        forbid: [
          ...["style", "className", "styles", "classNames", "vars"].map((propName) => ({
            propName,
            message: LOOK,
          })),
          ...["component", "renderRoot", "dangerouslySetInnerHTML"].map((propName) => ({
            propName,
            message: RENDER,
          })),
        ],
      },
    ],
  },
  overrides: [
    {
      files: ["src/web/components/*.ts", "src/web/components/*.tsx"],
      rules: { "no-restricted-imports": imports(LIBRARY, OWN_CSS, PAGES, APP) },
    },
    {
      files: ["src/web/pages/**", "src/web/hooks/**"],
      rules: { "no-restricted-imports": imports(LIBRARY, OWN_CSS, APP) },
    },
    // The two files that set up the app: the theme and the frame around every page.
    {
      files: ["src/web/app/Provider.tsx", "src/web/app/Shell.tsx"],
      rules: {
        "no-restricted-imports": imports(TABLER, OWN_CSS),
        "react/forbid-elements": "off",
        "react/forbid-dom-props": "off",
        "react/forbid-component-props": "off",
      },
    },
    // ui wraps Mantine, so the JSX rules stop here; it still keeps no CSS and knows nothing outside ui.
    {
      files: ["src/web/components/ui/**"],
      rules: {
        "no-restricted-imports": imports(OWN_CSS, OUTSIDE_UI),
        "react/forbid-elements": "off",
        "react/forbid-dom-props": "off",
        "react/forbid-component-props": "off",
        "nabu/no-conditional-color": "error",
      },
    },
    {
      files: ["src/web/components/ui/index.ts"],
      rules: { "no-restricted-imports": imports(OWN_CSS, OUTSIDE_UI, PASS_THROUGH) },
    },
  ],
});
