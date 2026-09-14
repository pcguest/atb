describe("ATB public homepage", () => {
  beforeEach(() => {
    cy.visit("/");
    cy.contains("h1", "Know what happened. Verify what was recorded.").should("be.visible");
  });

  it("presents the product, integration path, and evidence boundary", () => {
    cy.contains("Inside ATB View").should("be.visible");
    cy.get('img[alt*="ATB View investigating"]').should("be.visible");
    cy.contains("Use the interface that fits the workflow.").scrollIntoView().should("be.visible");
    cy.contains("Know what the evidence establishes.").scrollIntoView().should("be.visible");
  });

  it("passes the strict homepage accessibility audit", () => {
    cy.checkA11yStrict();
  });

  [1440, 1280, 1024, 900].forEach((width) => {
    it(`keeps the homepage within the ${width}px desktop viewport`, () => {
      cy.viewport(width, 900);
      cy.get("html").should(($html) => {
        expect($html[0].scrollWidth).to.be.at.most(width);
      });
      cy.screenshot(`homepage-${width}`, { capture: "fullPage" });
    });
  });
});
