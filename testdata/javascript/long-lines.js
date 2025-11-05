// File with long lines for testing max-len

const shortLine = "ok";

const veryLongLine = "This is a very long line that exceeds 80 characters and should be flagged by the max-len rule for ESLint validation purposes.";

function example() {
  const anotherVeryLongLineHereThatDefinitelyExceedsTheMaximumLengthAndShouldBeFlaggedByOurLengthEngine = true;
  return anotherVeryLongLineHereThatDefinitelyExceedsTheMaximumLengthAndShouldBeFlaggedByOurLengthEngine;
}

module.exports = { example };
