describe("ATB forensic investigation", () => {
  // Evidence status is deliberately bounded; it must not regress to a broad Trust claim.
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("follows the evidence-led primary flow", () => {
    cy.contains("What happened?").should("be.visible");
    cy.contains("One bounded finding requires review.").should("be.visible");

    cy.get('[aria-label="Investigation navigation"]').contains("button", "Findings").click();
    cy.contains("No matching approval").should("be.visible");
    cy.contains("button", "#2").click();
    // Selecting supporting evidence moves focus to its record, which may place the surface title
    // above the viewport; the evidence surface must still be mounted.
    cy.contains("Exact records").should("exist");

    cy.get('[aria-label="Investigation navigation"]').contains("button", "Evidence status").click();
    cy.contains("Is recorded evidence intact?").should("be.visible");
    cy.contains("Does the selected evidence profile pass?").should("be.visible");
    cy.contains("Is external or organisational custody recorded?").should("be.visible");
    cy.contains("ATB does not prove complete capture.").scrollIntoView().should("be.visible");
  });

  it("supports keyboard navigation with visible focus", () => {
    cy.get('[aria-label="Investigation navigation"] button').first().focus();
    cy.focused().should("have.css", "box-shadow").and("not.equal", "none");
  });

  it("passes the strict accessibility audit", () => {
    cy.checkA11yStrict();
  });
});
