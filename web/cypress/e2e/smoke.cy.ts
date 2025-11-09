describe('App Smoke Test', () => {
  it('displays the landing view', () => {
    cy.visit('/')
    cy.contains('Telegram To-Do Mini App').should('exist')
  })
})
