/**
 * Test file with length violations
 * Contains violations for line length, method length, and parameter count
 */
package com.example;

public class LengthViolations {

    // VIOLATION: Line length exceeds 100 characters
    private static final String VERY_LONG_CONSTANT_NAME_THAT_EXCEEDS_THE_MAXIMUM_LINE_LENGTH_AND_SHOULD_BE_FLAGGED = "test-value";

    // VIOLATION: Too many parameters (more than 4)
    public String processData(String firstName, String lastName, String email, String phone, String address, String city) {
        return firstName + " " + lastName + " - " + email;
    }

    // VIOLATION: Method is too long (more than 50 lines)
    public void veryLongMethod() {
        int line1 = 1;
        int line2 = 2;
        int line3 = 3;
        int line4 = 4;
        int line5 = 5;
        int line6 = 6;
        int line7 = 7;
        int line8 = 8;
        int line9 = 9;
        int line10 = 10;
        int line11 = 11;
        int line12 = 12;
        int line13 = 13;
        int line14 = 14;
        int line15 = 15;
        int line16 = 16;
        int line17 = 17;
        int line18 = 18;
        int line19 = 19;
        int line20 = 20;
        int line21 = 21;
        int line22 = 22;
        int line23 = 23;
        int line24 = 24;
        int line25 = 25;
        int line26 = 26;
        int line27 = 27;
        int line28 = 28;
        int line29 = 29;
        int line30 = 30;
        int line31 = 31;
        int line32 = 32;
        int line33 = 33;
        int line34 = 34;
        int line35 = 35;
        int line36 = 36;
        int line37 = 37;
        int line38 = 38;
        int line39 = 39;
        int line40 = 40;
        int line41 = 41;
        int line42 = 42;
        int line43 = 43;
        int line44 = 44;
        int line45 = 45;
        int line46 = 46;
        int line47 = 47;
        int line48 = 48;
        int line49 = 49;
        int line50 = 50;
        int line51 = 51;
        System.out.println("Line: " + line51);
    }

    public static void main(String[] args) {
        LengthViolations obj = new LengthViolations();
        obj.veryLongMethod();
    }
}
