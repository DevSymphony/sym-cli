# Test file that violates Python naming conventions

maxSize = 100  # 상수인데 소문자 (위반: UPPER_CASE 아님)
api_key = "secret123"  # 상수인데 snake_case (위반)

class user_profile:  # 클래스인데 snake_case (위반: PascalCase 아님)
    def __init__(self, userName):  # 매개변수 camelCase (위반)
        self.userName = userName  # 속성 camelCase (위반)

    def getUserName(self):  # 메서드 camelCase (위반: snake_case 아님)
        return self.userName

    def setUserName(self, newName):  # 메서드 camelCase (위반)
        self.userName = newName


class dataProcessor:  # 클래스 snake_case (위반)
    def processData(self, inputData):  # 메서드/매개변수 camelCase (위반)
        totalCount = 0  # 변수 camelCase (위반)
        for itemValue in inputData:  # 변수 camelCase (위반)
            totalCount += itemValue
        return totalCount


def calculateTotal(itemList):  # 함수 camelCase (위반)
    totalPrice = 0  # 변수 camelCase (위반)
    for itemPrice in itemList:  # 변수 camelCase (위반)
        totalPrice += itemPrice
    return totalPrice
