import { expect } from "chai";
import { Node1Client } from "../../src/thor-client";
import { SubscriptionBlockResponse } from "../../src/open-api-types-padded";

describe("GET /blocks/{revision}", function () {
  it("should be mining", async function () {
    const minedBlocks: SubscriptionBlockResponse[] = [];

    const { unsubscribe } = Node1Client.subscribeToBlocks((data) => {
      minedBlocks.push(data);
    });

    await new Promise((resolve) => setTimeout(resolve, 20_000));

    unsubscribe();

    expect(minedBlocks.length).to.be.greaterThan(1);
  });

  it("can get best block", async function () {
    const block = await Node1Client.getBlock("best");

    expect(block?.number).to.be.a("number");
    expect(block?.number).to.be.greaterThan(0);
  });
});
