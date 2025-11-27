// PMD violations test file
public class TestClass {
    private int myVar = 1;

    // PMD 위반: UnusedPrivateMethod (사용하지 않는 private 메서드)
    private void unusedMethod() {
        System.out.println("This method is never called");
    }

    public void testMethod() {
        final int CONST = 2;

        // PMD 위반: EmptyCatchBlock (빈 catch 블록)
        try {
            int result = 10 / CONST;
        } catch (Exception e) {
            // empty catch block - PMD should detect this
        }
    }

    // PMD 위반: CyclomaticComplexity (복잡도 > 10)
    public int complexMethod(int x) {
        if (x == 1) return 1;
        else if (x == 2) return 2;
        else if (x == 3) return 3;
        else if (x == 4) return 4;
        else if (x == 5) return 5;
        else if (x == 6) return 6;
        else if (x == 7) return 7;
        else if (x == 8) return 8;
        else if (x == 9) return 9;
        else if (x == 10) return 10;
        else if (x == 11) return 11;
        else return 0;
    }

    public int getMyVar() {
        return myVar;
    }
}
