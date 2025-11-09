import { render, screen } from '@testing-library/vue'

import App from './App.vue'

describe('App', () => {
  it('renders header content', () => {
    render(App, {
      global: {
        stubs: ['RouterView']
      }
    })

    expect(screen.getByText('Mini App 控制面板')).toBeInTheDocument()
  })
})
