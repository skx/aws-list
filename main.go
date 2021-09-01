//
// Export all running EC2 instances, along with brief details of them.
//
// Usage:
//
//    aws-list [-role=role-list.txt]
//
// Examples:
//
//  Export all instances in the current account:
//
//    $ aws-list
//
//  Export all instances, across the roles in the given text-file
//
//    $ aws-list -role=foo.list

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sts"
	"os"
	"strings"
	"time"
)

// Cache of creation-time/date
var cache map[string]string

// Get the creation-date of the given AMI.
//
// Values are cached.
func amiCreation(svc *ec2.EC2, id string) (string, error) {

	// Lookup in the cache to see if we've already found the creation
	// date for this AMI
	cached, ok := cache[id]
	if ok {
		return cached, nil
	}

	// Setup a filter for the AMI we're looking for.
	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(id),
		},
	}

	// Run the search
	result, err := svc.DescribeImages(input)
	if err != nil {
		// Message from an error.
		return "", fmt.Errorf("error getting image info: %s", err.Error())
	}

	// If we got a result then we can return the creation time
	// (as a string)
	if len(result.Images) > 0 {

		// But save in a cache for the future
		date := *result.Images[0].CreationDate
		cache[id] = date
		return date, nil
	}
	return "", fmt.Errorf("no date for %s", id)
}

// Sync from remote to local
func Sync(svc *ec2.EC2, acct string) error {

	// Get the instances which are running/pending
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running"), aws.String("pending")},
			},
		},
	}

	// Create new EC2 client
	result, err := svc.DescribeInstances(params)
	if err != nil {
		return fmt.Errorf("DescribeInstances failed: %s", err)
	}

	// For each instance show stuff
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {

			// We have a running EC2 instnace.

			// Collect the data we want
			id := *instance.InstanceId

			// Find the name.
			name := *instance.InstanceId

			// Look for the name, which is set via a Tag.
			i := 0
			for i < len(instance.Tags) {

				if *instance.Tags[i].Key == "Name" {
					name = *instance.Tags[i].Value
				}
				i++
			}

			// AMI name
			ami := *instance.ImageId

			//
			// Get the AMI creation-date
			//
			create, err := amiCreation(svc, ami)
			if err != nil {
				return fmt.Errorf("failed to get creation date of %s: %s", ami, err.Error())
			}

			//
			// Parse the date, so we can report how many days
			// ago the AMI was created.
			//
			t, err := time.Parse("2006-01-02T15:04:05.000Z", create)
			if err != nil {
				return fmt.Errorf("failed to parse time string %s: %s", create, err)
			}

			//
			// Count how old the AMI is in days
			//
			date := time.Now()
			diff := date.Sub(t)
			create = fmt.Sprintf("%d days", (int(diff.Hours() / 24)))

			//
			// Now show all the information
			//
			fmt.Printf("%s %s %s %s %s\n", acct, id, name, ami, create)
		}
	}
	return nil
}

func main() {

	//
	// Create our cache
	//
	cache = make(map[string]string)

	//
	// Command-line flags
	//
	rolePtr := flag.String("roles", "", "Specify the path to a file containing AWS-roles to assume, one per line")

	//
	// Parse flags
	//
	flag.Parse()

	//
	// Get the connection, using default creds
	//
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-central-1"),
	})
	if err != nil {
		fmt.Printf("AWS login failed: %s\n", err.Error())
		return
	}

	//
	// Create a new session to find our account
	//
	stsSvc := sts.New(sess)
	out, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Printf("Failed to get identity: %s", err.Error())
		return
	}

	acct := *out.Account

	//
	// If we have no role-list then just dump our current account
	//
	if *rolePtr == "" {

		svc := ec2.New(sess)

		err := Sync(svc, acct)
		if err != nil {
			fmt.Printf("error syncing account %s\n", err.Error())
			return
		}

		return
	}

	//
	// OK we have a list of roles, handle them one by one
	//
	file, err := os.Open(*rolePtr)
	if err != nil {
		fmt.Printf("Error opening role-file: %s %s\n", *rolePtr, err.Error())
		return
	}
	defer file.Close()

	//
	// Process the role-file line by line
	//
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		// Get the line
		role := scanner.Text()

		// Skip comments
		if strings.HasPrefix(role, "#") {
			continue
		}

		// process
		creds := stscreds.NewCredentials(sess, role)

		// Create service client value configured for credentials
		// from assumed role.
		svc := ec2.New(sess, &aws.Config{Credentials: creds})

		// We'll get the account from the string which looks like this:
		//
		// arn:aws:iam::1234:role/blah-abc
		//
		// We split by ":" and get the fourth field.
		//
		data := strings.Split(role, ":")
		acct := data[4]

		// Process the running instances
		err = Sync(svc, acct)
		if err != nil {
			fmt.Printf("Error for role %s %s\n", role, err.Error())
		}
	}

	//
	// Error processing the end of the file?
	//
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error processing role-file: %s %s\n", *rolePtr, err.Error())
		return
	}

}
