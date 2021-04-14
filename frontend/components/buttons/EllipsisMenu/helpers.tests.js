import { calculateTooltipDirection } from "./helpers";

describe("EllipsisMenu - helpers", () => {
  describe("#calculateTooltipDirection", () => {
    it('returns "left" if the element does not fit to the right in the browser', () => {
      const el = {
        getBoundingClientRect: () => {
          return {
            // test DOM window.innerWidth is 1024px
            right: 725,
          };
        },
      };

      expect(calculateTooltipDirection(el)).toEqual("left");
    });

    it('returns "right" if the element fits to the right in the browser', () => {
      const el = {
        getBoundingClientRect: () => {
          return {
            // test DOM window.innerWidth is 1024px
            right: 724,
          };
        },
      };

      expect(calculateTooltipDirection(el)).toEqual("right");
    });
  });
});
