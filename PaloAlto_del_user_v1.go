package main

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var users_list []string
var filePath = string("./users")
var apiKey string

// create a structure of an xml document
type Response struct {
	XMLName xml.Name `xml:"response"`
	Result  struct {
		Entry []struct {
			Name_grp []string `xml:"name,attr"`
			RuleName []struct {
				Name        string   `xml:"name,attr"`
				To_zone     []string `xml:"to>member"`
				From_zone   []string `xml:"from>member"`
				Source_addr []string `xml:"source>member"`
				Source_user []string `xml:"source-user>member"`
				App         []string `xml:"application>member"`
				Service     []string `xml:"service>member"`
				Dst_addr    []string `xml:"destination>member"`
			} `xml:"pre-rulebase>security>rules>entry"`
		} `xml:"entry"`
	} `xml:"result>config>devices>entry>device-group"`
}

// create url request to Panorama
func ask_api_key() string {
	fmt.Println("Enter Your apikey: ")
	fmt.Scan(&apiKey)
	url1 := "https://x.x.x.x/api/?type=op&cmd=<show><config><running></running></config></show>&key="
	url2 := apiKey
	url := url1 + url2
	return url
}

// read config from Panorama
func read_config(x string) string {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(x)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	return string(body)
}

// get users list for deletion from the file
func get_users(x string) []string {
	file, err := os.Open(x)
	if err != nil {
		fmt.Printf("Could not open the file due to this %s error \n", err)
	}
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		b := strings.TrimSpace(fileScanner.Text())
		users_list = append(users_list, b)
	}
	return users_list
}

// decompose XML config of the panorama using Response struct
func unmarshal_xml(data string) Response {
	//create an empty slice using "Response" structure
	d := Response{}
	//decode the xml config and write the result to "d" slice
	if err := xml.Unmarshal([]byte(data), &d); err != nil {
		panic(err)
	}
	return d

}

// Check how many users from delete list match for this Panorama security-rule users list
func search_user(list_of_users_in_rule []string, list_of_users_for_del []string) []string {
	Match_user := []string{}
	for _, del_user := range list_of_users_for_del {
		for _, rule_user := range list_of_users_in_rule {
			if rule_user == del_user {
				Match_user = append(Match_user, del_user)
			}
		}
	}
	return Match_user
}

func main() {
	url := ask_api_key()
	decoded_config := unmarshal_xml(read_config(url))
	delete_users_list := get_users(filePath)
	for _, dev_config := range decoded_config.Result.Entry { //iterate through device-config
		if dev_config.Name_grp[0] == "fw-vpn-pa" {
			for _, Rule := range dev_config.RuleName { //iterate through security rules names
				rule_users_list := Rule.Source_user                                    //list of users inside the security-rule
				Name_of_the_rule := Rule.Name                                          //Name of the security rule
				found_users_for_del := search_user(rule_users_list, delete_users_list) // check if we have users for deletion inside this security-rule
				if len(found_users_for_del) == len(rule_users_list) {
					fmt.Printf("\n delete device-group %+v pre-rulebase security rules %+v", dev_config.Name_grp[0], Name_of_the_rule) //delete security rule
				} else if len(found_users_for_del) != len(rule_users_list) && len(found_users_for_del) != 0 {
					for _, found_user := range found_users_for_del {
						fmt.Printf("\n delete device-group %+v pre-rulebase security rules %+v source-user %+v", dev_config.Name_grp[0], Name_of_the_rule, found_user) //delete user from the security rule
					}
				}
			}
		}
	}
	fmt.Printf("\n\n")
}
