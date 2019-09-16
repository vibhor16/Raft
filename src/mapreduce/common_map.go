package mapreduce

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
)

func doMap(
	jobName string, // the name of the MapReduce job
	mapTask int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(filename string, contents string) []KeyValue,
) {

	//
	// doMap manages one map task: it should read one of the input files
	// (inFile), call the user-defined map function (mapF) for that file's
	// contents, and partition mapF's output into nReduce intermediate files.
	//
	// There is one intermediate file per reduce task. The file name
	// includes both the map task number and the reduce task number. Use
	// the filename generated by reduceName(jobName, mapTask, r)
	// as the intermediate file for reduce task r. Call ihash() (see
	// below) on each key, mod nReduce, to pick r for a key/value pair.
	//
	// mapF() is the map function provided by the application. The first
	// argument should be the input file name, though the map function
	// typically ignores it. The second argument should be the entire
	// input file contents. mapF() returns a slice containing the
	// key/value pairs for reduce; see common.go for the definition of
	// KeyValue.
	//
	// Look at Go's ioutil and os packages for functions to read
	// and write files.
	//
	// Coming up with a scheme for how to format the key/value pairs on
	// disk can be tricky, especially when taking into account that both
	// keys and values could contain newlines, quotes, and any other
	// character you can think of.
	//
	// One format often used for serializing data to a byte stream that the
	// other end can correctly reconstruct is JSON. You are not required to
	// use JSON, but as the output of the reduce tasks *must* be JSON,
	// familiarizing yourself with it here may prove useful. You can write
	// out a data structure as a JSON string to a file using the commented
	// code below. The corresponding decoding functions can be found in
	// common_reduce.go.
	//
	//	   enc := json.NewEncoder(file)
	//	   for _, kv := ... {
	//	     err := enc.Encode(&kv)
	//
	// Remember to close the file after you have written all the values!
	//

	// Reading the input content as bytes
	byteContent, err := ioutil.ReadFile(inFile)
	if err != nil {
		fmt.Print(err)
	}

	// Creating mapped content to be written to the intermediate files
	mappedContent := mapF(inFile, string(byteContent))

	// Creating/opening intermediate files
	var intermediateFilesMap = make(map[string]*os.File)
	for _, value := range mappedContent {
		intermediateFileName := reduceName(jobName, mapTask, ihash(value.Key)%nReduce)
		if _, fileExists := intermediateFilesMap[intermediateFileName]; !fileExists {
			fmt.Printf("File name: [%s]\n", intermediateFileName)
			intermediateFile, err := os.OpenFile(intermediateFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Print(err)
			}

			intermediateFilesMap[intermediateFileName] = intermediateFile
		}
	}

	// Writing the mapped content into the intermediate files
	for _, value := range mappedContent {
		intermediateFileName := reduceName(jobName, mapTask, ihash(value.Key)%nReduce)
		intermediateFile := intermediateFilesMap[intermediateFileName]

		// Encoding the mapped result in JSON format
		enc := json.NewEncoder(intermediateFile)
		err = enc.Encode(KeyValue{value.Key, value.Value})
		if err != nil {
			fmt.Println(err)
		}

		if err != nil {
			fmt.Println(err)
		}
	}

	// Closing all the files
	for _, value := range mappedContent {
		intermediateFileName := reduceName(jobName, mapTask, ihash(value.Key)%nReduce)
		err := intermediateFilesMap[intermediateFileName].Close()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func ihash(s string) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		fmt.Println(err)
	}
	return int(h.Sum32() & 0x7fffffff)
}
