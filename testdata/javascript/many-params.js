// File with functions that have too many parameters

function tooManyParams(a, b, c, d, e, f, g) {
  return a + b + c + d + e + f + g;
}

const arrowWithManyParams = (x1, x2, x3, x4, x5, x6, x7, x8) => {
  return x1 + x2 + x3 + x4 + x5 + x6 + x7 + x8;
};

function goodFunction(a, b, c) {
  return a + b + c;
}

module.exports = { tooManyParams, arrowWithManyParams, goodFunction };
