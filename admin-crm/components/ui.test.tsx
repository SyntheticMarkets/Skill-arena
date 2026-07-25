import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmptyState, ErrorState, Modal } from "./ui";

describe("shared operational states", () => {
  it("renders accessible empty and error states", () => {
    const retry = vi.fn();
    const { rerender } = render(<EmptyState title="Queue clear" description="No work remains." />);
    expect(screen.getByRole("heading", { name: "Queue clear" })).toBeVisible();
    rerender(<ErrorState message="Service unavailable." retry={retry} />);
    screen.getByRole("button", { name: "Try again" }).click();
    expect(retry).toHaveBeenCalledOnce();
  });

  it("labels modal dialogs and close controls", () => {
    render(<Modal title="Review withdrawal" onClose={vi.fn()}><p>Decision</p></Modal>);
    expect(screen.getByRole("dialog", { name: "Review withdrawal" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Close dialog" })).toBeVisible();
  });
});
