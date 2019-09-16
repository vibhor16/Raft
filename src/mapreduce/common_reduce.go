package mapreduce

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTask int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	//
	// doReduce manages one reduce task: it should read the intermediate
	// files for the task, sort the intermediate key/value pairs by key,
	// call the user-defined reduce function (reduceF) for each key, and
	// write reduceF's output to disk.
	//
	// You'll need to read one intermediate file from each map task;
	// reduceName(jobName, m, reduceTask) yields the file
	// name from map task m.
	//
	// Your doMap() encoded the key/value pairs in the intermediate
	// files, so you will need to decode them. If you used JSON, you can
	// read and decode by creating a decoder and repeatedly calling
	// .Decode(&kv) on it until it returns an error.
	//
	// You may find the first example in the golang sort package
	// documentation useful.
	//
	// reduceF() is the application's reduce function. You should
	// call it once per distinct key, with a slice of all the values
	// for that key. reduceF() returns the reduced value for that key.
	//
	// You should write the reduce output as JSON encoded KeyValue
	// objects to the file named outFile. We require you to use JSON
	// because that is what the merger than combines the output
	// from all the reduce tasks expects. There is nothing special about
	// JSON -- it is just the marshalling format we chose to use. Your
	// output code will look something like this:
	//
	// enc := json.NewEncoder(file)
	// for key := ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()
	//
	// Your code here (Part I).
	//

	// Contains the reduced content for each key and value pair
	combineMap := make(map[string][]string)
	// Contains only the keys of the combineMap to be sorted
	var combineMapkeys []int

	// Opening intermediate files
	var intermediateFileMap = make(map[string]*os.File)
	for i := 0; i < nMap; i++ {
		intermediateFileName := reduceName(jobName, i, reduceTask)
		if _, fileExists := intermediateFileMap[intermediateFileName]; !fileExists {
			intermediateFile, err := os.Open(intermediateFileName)
			if err != nil {
				fmt.Print(err)
			}
			intermediateFileMap[intermediateFileName] = intermediateFile
		}
	}

	// Creating output file
	outputFile, err := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}

	// Reducing the intermediate files to a single output file
	for i := 0; i < nMap; i++ {
		intermediateFileName := reduceName(jobName, i, reduceTask)
		intermediateFile := intermediateFileMap[intermediateFileName]

		// Decoding the JSON content of the intermediate files
		decoder := json.NewDecoder(intermediateFile)
		for {
			var entry KeyValue
			if err := decoder.Decode(&entry); err == io.EOF {
				break
			} else if err != nil {
				fmt.Println(err)
			}
			combineMap[entry.Key] = append(combineMap[entry.Key], entry.Value)
		}
	}

	// Creating a list of keys converted from string to int
	for key := range combineMap {
		intKey, err := strconv.Atoi(key)
		if err != nil {
			fmt.Println(err)
		}

		combineMapkeys = append(combineMapkeys, intKey)
	}

	// Sorting the keys  in ascending order
	sort.Ints(combineMapkeys)

	// Writing the reduced result in the outFile
	for _, key := range combineMapkeys {
		var entry KeyValue
		stringKey := strconv.Itoa(key)

		entry.Key = stringKey
		entry.Value = reduceF(stringKey, combineMap[stringKey])

		// Encoding the reduced result in JSON format in output file
		enc := json.NewEncoder(outputFile)
		err = enc.Encode(entry)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Closing intermediate files
	for i := 0; i < nMap; i++ {
		intermediateFileName := reduceName(jobName, i, reduceTask)
		err := intermediateFileMap[intermediateFileName].Close()
		if err != nil {
			fmt.Println(err)
		}
	}

	// Closing the output files
	err = outputFile.Close()
	if err != nil {
		fmt.Println(err)
	}
}
