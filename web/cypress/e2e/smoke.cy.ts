describe('App Smoke Test', () => {
  it('displays the landing view', () => {
    cy.visit('/')
    cy.contains('待办任务').should('exist')
    cy.contains('Pending').should('exist')
  })
})
