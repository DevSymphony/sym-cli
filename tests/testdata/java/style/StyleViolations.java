/**
 * Test file with style violations
 * Contains violations for indentation, spacing, and formatting
 */
package com.example;

public class StyleViolations {

// VIOLATION: Missing indentation for class member
private String name;

    // VIOLATION: Inconsistent indentation
    public void badIndentation() {
    int x = 1;
      int y = 2;
        int z = 3;
      System.out.println(x + y + z);
    }

    // VIOLATION: Opening brace on next line (Java convention: same line)
    public void badBracePlacement()
    {
        System.out.println("Bad brace placement");
    }

    // VIOLATION: Multiple statements on one line
    public void multipleStatements() { int a = 1; int b = 2; System.out.println(a + b); }

    // VIOLATION: No space after if/for keywords
    public void noSpaceAfterKeyword() {
        if(true){
            for(int i=0;i<10;i++){
                System.out.println(i);
            }
        }
    }

    // VIOLATION: Inconsistent spacing around operators
    public int badOperatorSpacing() {
        int result=10+20*30/5-2;
        return result;
    }

    // VIOLATION: Long line exceeding typical style guide limit
    private static final String EXTREMELY_LONG_LINE_THAT_EXCEEDS_REASONABLE_LENGTH_LIMITS_AND_SHOULD_BE_WRAPPED_OR_REFACTORED = "value";

    // VIOLATION: Missing blank line before method
    public void missingBlankLine() {
        System.out.println("No blank line above");
    }
    public void anotherMethod() {
        System.out.println("Methods too close together");
    }
}
