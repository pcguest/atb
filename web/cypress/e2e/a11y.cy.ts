describe("ATB investigation accessibility", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("passes strict accessibility audit", () => {
    cy.checkA11yStrict();
  });

  it("passes strict accessibility audit in light theme", () => {
    cy.get('select[aria-label="Theme"]').select("light");
    cy.checkA11yStrict();
  });
});
