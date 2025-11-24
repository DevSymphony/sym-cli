/**
 * Test file with valid Java style
 * Follows standard Java formatting conventions
 */
package com.example;

public class ValidStyle {

    private String name;
    private int value;

    public ValidStyle() {
        this.name = "default";
        this.value = 0;
    }

    public void properIndentation() {
        int x = 1;
        int y = 2;
        int z = 3;
        System.out.println(x + y + z);
    }

    public void properBracePlacement() {
        System.out.println("Correct brace placement");
    }

    public void properSpacing() {
        if (true) {
            for (int i = 0; i < 10; i++) {
                System.out.println(i);
            }
        }
    }

    public int properOperatorSpacing() {
        int result = 10 + 20 * 30 / 5 - 2;
        return result;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public int getValue() {
        return value;
    }

    public void setValue(int value) {
        this.value = value;
    }

    public static void main(String[] args) {
        ValidStyle obj = new ValidStyle();
        obj.properIndentation();
        obj.properSpacing();
        System.out.println("Result: " + obj.properOperatorSpacing());
    }
}
