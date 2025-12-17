import type { Config } from "tailwindcss";

import daisyui from "daisyui";

const config: Config = {
  content: ["./index.html", "./src/**/*.{vue,ts,js,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [daisyui],
  daisyui: {
    themes: ["business", "black"], // Dark themes
    base: true,
    darkTheme: "business",
  },
};

export default config;
