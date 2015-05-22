/*
 * Minio Client, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"errors"
	"io"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

func runCatCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cat", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Fatalf("Unable to read config file ‘%s’. Reason: %s.\n", mustGetMcConfigPath(), iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Fatalf("Unknown type of URL ‘%s’. Reason: %s.\n", e.url, e)
		default:
			console.Fatalf("Unable to parse arguments. Reason: %s.\n", iodine.ToError(err))
		}
	}

	sourceURLs := urls
	sourceURLConfigMap, err := getHostConfigs(sourceURLs)
	if err != nil {
		console.Fatalf("Unable to read host configuration for ‘%s’ from config file ‘%s’. Reason: %s.\n",
			sourceURLs, mustGetMcConfigPath(), iodine.ToError(err))
	}
	humanReadable, err := doCatCmd(sourceURLConfigMap)
	if err != nil {
		console.Fatalln(humanReadable)
	}
}

func doCatCmd(sourceURLConfigMap map[string]*hostConfig) (string, error) {
	for url, config := range sourceURLConfigMap {
		sourceClnt, err := getNewClient(url, config)
		if err != nil {
			return "Unable to create client: " + url, iodine.New(err, nil)
		}
		reader, size, err := sourceClnt.GetObject(0, 0)
		if err != nil {
			return "Unable to retrieve file: " + url, iodine.New(err, nil)
		}
		defer reader.Close()
		_, err = io.CopyN(os.Stdout, reader, int64(size))
		if err != nil {
			switch e := iodine.ToError(err).(type) {
			case *os.PathError:
				return "Reading data to stdout failed, unexpected problem.. please report this error", iodine.New(e, nil)
			default:
				return "Reading data from source failed: " + url, iodine.New(errors.New("Copy data from source failed"), nil)
			}
		}
	}
	return "", nil
}
