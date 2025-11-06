// File with style violations

// Bad: inconsistent indentation
function badIndentation() {
const x = 1;
  const y = 2;
    const z = 3;
  return x + y + z;
}

// Bad: double quotes (should be single)
const message = "Hello World";

// Bad: missing semicolons
const a = 1
const b = 2
const c = 3

// Bad: long line exceeding 100 characters
const veryLongLineHereThisIsWayTooLongAndShouldBeReportedByTheLinterAsAViolationOfTheLineLength = true;

// Bad: multiple statements on one line
const x = 1; const y = 2; const z = 3;

// Good examples
function goodIndentation() {
  const x = 1;
  const y = 2;
  const z = 3;
  return x + y + z;
}
