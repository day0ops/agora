import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { IsolationCheck } from "./IsolationCheck";
import * as api from "../api";

describe("IsolationCheck", () => {
  it("shows the real agent reply after probing trust-safety", async () => {
    vi.spyOn(api, "postIsolationCheck").mockResolvedValue({
      reply: "I don't have access to order history.",
    });
    render(<IsolationCheck />);
    fireEvent.click(
      screen.getByRole("button", { name: /probe trust-safety/i }),
    );
    await waitFor(() =>
      expect(
        screen.getByText(/don't have access to order history/i),
      ).toBeInTheDocument(),
    );
    expect(api.postIsolationCheck).toHaveBeenCalledWith("trust-safety");
  });

  it("shows an error message when the probe call fails", async () => {
    vi.spyOn(api, "postIsolationCheck").mockRejectedValue(
      new api.ApiError(502, "isolation probe call failed"),
    );
    render(<IsolationCheck />);
    fireEvent.click(screen.getByRole("button", { name: /probe auto-bidder/i }));
    await waitFor(() =>
      expect(
        screen.getByText(/isolation probe call failed/i),
      ).toBeInTheDocument(),
    );
  });
});
