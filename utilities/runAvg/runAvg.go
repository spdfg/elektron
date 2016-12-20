/*
A utility to calculate the running average.

One should implement Val() to be able to use this utility.
*/

package runAvg

import (
	"errors"
	"container/list"
)

type Interface interface {
	// Value to use for running average calculation.
	Val() float64
	// Unique ID
	ID() string
}

type runningAverageCalculator struct {
	window list.List
	windowSize int
	currentSum float64
	numberOfElementsInWindow int
}

// singleton instance
var racSingleton *runningAverageCalculator

// return single instance
func getInstance(curSum float64, n int, wSize int) *runningAverageCalculator {
	if racSingleton == nil {
		racSingleton = &runningAverageCalculator {
			windowSize: wSize,
			currentSum: curSum,
			numberOfElementsInWindow: n,
		}
		return racSingleton
	} else {
		// Updating window size if a new window size is given.
		if wSize != racSingleton.windowSize {
			racSingleton.windowSize = wSize
		}
		return racSingleton
	}
}

// Compute the running average by adding 'data' to the window.
// Updating currentSum to get constant time complexity for every running average computation.
func (rac *runningAverageCalculator) calculate(data Interface) float64 {
	if rac.numberOfElementsInWindow < rac.windowSize {
		rac.window.PushBack(data)
		rac.numberOfElementsInWindow++
		rac.currentSum += data.Val()
	} else {
		// removing the element at the front of the window.
		elementToRemove := rac.window.Front()
		rac.currentSum -= elementToRemove.Value.(Interface).Val()
		rac.window.Remove(elementToRemove)
		
		// adding new element to the window
		rac.window.PushBack(data)
		rac.currentSum += data.Val()
	}
	return rac.currentSum / float64(rac.window.Len())
}

/*
If element with given ID present in the window, then remove it and return (removeElement, nil).
Else, return (nil, error)
*/
func (rac *runningAverageCalculator) removeFromWindow(id string) (interface{}, error) {
	for element := rac.window.Front(); element != nil; element = element.Next() {
		if elementToRemove := element.Value.(Interface); elementToRemove.ID() == id {
			rac.window.Remove(element)
			return elementToRemove, nil
		}
	}
	return nil, errors.New("Error: Element not found in the window.")
}

// Taking windowSize as a parameter to allow for sliding window implementation.
func Calc(data Interface, windowSize int) float64 {
	rac := getInstance(0.0, 0, windowSize)
	return rac.calculate(data)
}

// Remove element from the window if it is present.
func Remove(id string) (interface{}, error) {
	// checking if racSingleton has been instantiated
	if racSingleton == nil {
		return nil, errors.New("Error: Not instantiated. Please call Init() to instantiate.")
	} else {
		return racSingleton.removeFromWindow(id)
	}
}

// initialize the parameters of the running average calculator
func Init() {
	// checking to see if racSingleton needs top be instantiated
	if racSingleton == nil {
		racSingleton = getInstance(0.0, 0, 0)
	}
	// Setting parameters to default values. Could also set racSingleton to nil but this leads to unnecessary overhead of creating
	// another instance when Calc is called.
	racSingleton.window.Init()
	racSingleton.windowSize = 0
	racSingleton.currentSum = 0.0
	racSingleton.numberOfElementsInWindow = 0
}