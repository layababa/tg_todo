import type { Config } from "tailwindcss";

import daisyui from "daisyui";

const config: Config = {
  content: ["./index.html", "./src/**/*.{vue,ts,js,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        display: ["Space Grotesk", "sans-serif"],
        mono: ["JetBrains Mono", "monospace"],
        sc: ["Noto Sans SC", "sans-serif"],
      },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        // OKX / Web3 Cyberpunk Dark Theme
        tgtodo: {
          primary: "#ABF600", // Neon green
          "primary-content": "#000000", // Black text on primary
          secondary: "#333333", // Border dim
          "secondary-content": "#FFFFFF",
          accent: "#ABF600", // Same as primary for consistency
          "accent-content": "#000000",
          neutral: "#0A0A0A", // Panel background
          "neutral-content": "#FFFFFF",
          "base-100": "#000000", // Main background - pure black
          "base-200": "#0A0A0A", // Panel background
          "base-300": "#1A1A1A", // Slightly lighter
          "base-content": "#FFFFFF", // White text
          info: "#ABF600",
          success: "#ABF600",
          warning: "#FFB300",
          error: "#FF0055",
          // Custom CSS variables for additional styling
          "--rounded-box": "0", // Sharp corners for tech feel
          "--rounded-btn": "0",
          "--rounded-badge": "0",
          "--animation-btn": "0.2s",
          "--animation-input": "0.2s",
          "--btn-focus-scale": "0.98",
          "--border-btn": "1px",
          "--tab-border": "1px",
          "--tab-radius": "0",
        },
      },
    ],
    base: true,
    darkTheme: "tgtodo",
  },
};

export default config;
