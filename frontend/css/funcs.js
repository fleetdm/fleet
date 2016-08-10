/**
 * Global CSS Functions.
 * @module css/funcs
 */
module.exports = {
  /**
  * Returns a string
  * @param {...number} val - A positive or negative number.
  * @example
  * // returns "height: 5px;"
    height: px(5);
  */
  px: function(val) {
    return val + 'px';
  }
};
