describe("ATB Trust Dashboard", () => {
  it("load bundle -> display Viewer Health", () => {
    cy.visit("/view/");
    cy.get('[data-testid="viewer-health-value"]').should("be.visible");
    cy.contains("Verify bundle").should("be.visible");
  });
});
