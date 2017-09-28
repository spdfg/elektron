/*
A utility to calculate the running average.

One should implement Val() and ID() to use this utility.
*/

package runAvg

import (
	"container/list"
	"errors"
)

type Interface interface {
	// Value to use for running average calculation.
	Val() float64
	// Unique ID.
	ID() string
}

type runningAverageCalculator struct {
	considerationWindow     list.List
	considerationWindowSize int
	currentSum              float64
}

// Singleton instance.
var racSingleton *runningAverageCalculator

// Return single instance.
func getInstance(curSum float64, wSize int) *runningAverageCalculator {
	if racSingleton == nil {
		racSingleton = &runningAverageCalculator{
			considerationWindowSize: wSize,
			currentSum:              curSum,
		}
		return racSingleton
	} else {
		// Updating window size if a new window size is given.
		if wSize != racSingleton.considerationWindowSize {
			racSingleton.considerationWindowSize = wSize
		}
		return racSingleton
	}
}

// Compute the running average by adding 'data' to the window.
// Updating currentSum to get constant time complexity for every running average computation.
func (rac *runningAverageCalculator) calculate(data Interface) float64 {
	if rac.considerationWindow.Len() < rac.considerationWindowSize {
		rac.considerationWindow.PushBack(data)
		rac.currentSum += data.Val()
	} else {
		// Removing the element at the front of the window.
		elementToRemove := rac.considerationWindow.Front()
		rac.currentSum -= elementToRemove.Value.(Interface).Val()
		rac.considerationWindow.Remove(elementToRemove)

		// Adding new element to the window.
		rac.considerationWindow.PushBack(data)
		rac.currentSum += data.Val()
	}
	return rac.currentSum / float64(rac.considerationWindow.Len())
}

/*
If element with given ID present in the window, then remove it and return (removeElement, nil).
Else, return (nil, error).
*/
func (rac *runningAverageCalculator) removeFromWindow(id string) (interface{}, error) {
	for element := rac.considerationWindow.Front(); element != nil; element = element.Next() {
		if elementToRemove := element.Value.(Interface); elementToRemove.ID() == id {
			rac.considerationWindow.Remove(element)
			rac.currentSum -= elementToRemove.Val()
			return elementToRemove, nil
		}
	}
	return nil, errors.New("Error: Element not found in the window.")
}

// Taking windowSize as a parameter to allow for sliding window implementation.
func Calc(data Interface, windowSize int) float64 {
	rac := getInstance(0.0, windowSize)
	return rac.calculate(data)
}

// Remove element from the window if it is present.
func Remove(id string) (interface{}, error) {
	// Checking if racSingleton has been instantiated.
	if racSingleton == nil {
		return nil, errors.New("Error: Not instantiated. Please call Init() to instantiate.")
	} else {
		return racSingleton.removeFromWindow(id)
	}
}

// Initialize the parameters of the running average calculator.
func Init() {
	// Checking to see if racSingleton needs top be instantiated.
	if racSingleton == nil {
		racSingleton = getInstance(0.0, 0)
	}
	// Setting parameters to default values. Could also set racSingleton to nil but this leads to unnecessary overhead of creating
	// another instance when Calc is called.
	racSingleton.considerationWindow.Init()
	racSingleton.considerationWindowSize = 0
	racSingleton.currentSum = 0.0
}
