package utils

func RemoveEmptyStringsFromArray(array []string) []string {
    var newArray []string
    for _, item := range array {
        if item != "" {
            newArray = append(newArray, item)
        }
    }
    return newArray
}
