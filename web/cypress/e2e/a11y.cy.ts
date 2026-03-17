describe("Trust Dashboard accessibility", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("passes strict accessibility audit", () => {
    cy.checkA11yStrict();
  });
});
