import type { Config } from 'tailwindcss'

import daisyui from 'daisyui'

const config: Config = {
  content: ['./index.html', './src/**/*.{vue,ts,js,tsx}'],
  theme: {
    extend: {}
  },
  plugins: [daisyui],
  daisyui: {
    themes: ['lemonade'],
    base: true,
    darkTheme: 'lemonade'
  }
}

export default config
