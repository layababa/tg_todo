const DEBUG_INIT_DATA =
  'query_id=debug&user=%7B%22id%22%3A999%2C%22first_name%22%3A%22Debug%22%7D&auth_date=1700000000&hash=dummy'

describe('Task API integration', () => {
  it('renders tasks from API and toggles status with correct headers', () => {
    cy.intercept('GET', 'http://localhost:8080/tasks', { fixture: 'tasks.json' }).as('fetchTasks')

    cy.visit('/', {
      onBeforeLoad(win) {
        win.localStorage.setItem('tg_todo_debug_init_data', DEBUG_INIT_DATA)
      }
    })

    cy.wait('@fetchTasks')

    cy.get('[data-cy="task-card"]').should('have.length', 2)
    cy.get('[data-cy="task-card"]').first().contains('撰写 API 文档')

    cy.fixture('tasks.json').then(tasks => {
      const updated = { ...tasks[0], status: 'completed' }
      cy.intercept('PATCH', 'http://localhost:8080/tasks/1', req => {
        expect(req.headers['x-telegram-init-data']).to.equal(DEBUG_INIT_DATA)
        expect(req.body).to.deep.equal({ status: 'completed' })
        req.reply({ statusCode: 200, body: updated })
      }).as('toggleTask')

      cy.get('[data-cy="task-card"]')
        .first()
        .find('input[type="checkbox"]')
        .click({ force: true })

      cy.wait('@toggleTask')
      cy.get('[data-cy="task-card"]').first().contains('已完成').should('exist')
    })
  })
})
