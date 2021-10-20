// Copyright 2017-2020 Dale Farnsworth. All rights reserved.

// Dale Farnsworth
// 1007 W Mendoza Ave
// Mesa, AZ  85210
// USA
//
// dale@farnsworth.org

// This file is part of Radio.
//
// Radio is free software: you can redistribute it and/or modify
// it under the terms of version 3 of the GNU Lesser General Public
// License as published by the Free Software Foundation.
//
// Radio is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Radio.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dalefarnsworth-dmr/codeplug"
	"github.com/dalefarnsworth-dmr/debug"
	"github.com/dalefarnsworth-dmr/dfu"
	"github.com/dalefarnsworth-dmr/userdb"
)

func errorf(s string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, s, v...)
}

func usage() {
	subCommandUsages := []string{
		"codeplugToJSON <codeplugFile> <jsonFile>",
		"codeplugToText <codeplugFile> <textFile>",
		"codeplugToXLSX <codeplugFile> <xlsxFile>",
		"countryCounts <usersFile>",
		"filterUsers <countriesFile> <inUsersFile> <outUsersFile>",
		"getMergedUsers <usersFile>",
		"getAbbreviatedUsers <usersFile>",
		"getUsers <usersFile>",
		"jsonToCodeplug <jsonFile> <codeplugFile>",
		"newCodeplug -model <model> -freq <freqRange> <codeplugFile>",
		"readCodeplug -model <model> -freq <freqRange> <codeplugFile>",
		"readMD380Users <usersFile>",
		"readSPIFlash <filename>",
		"textToCodeplug <textFile> <codeplugFile>",
		"userCountries <usersFile> <countriesFile>",
		"version",
		"writeCodeplug <codeplugFile>",
		"writeMD380Firmware <firmwareFile>",
		"writeMD2017Users <usersFile>",
		"writeMD380Users <usersFile>",
		"writeUV380Users <usersFile>",
		"xlsxToCodeplug <xlsxFile> <codeplugFile>",
	}

	errorf("Usage %s <subCommand> args\n", os.Args[0])
	errorf("subCommands:\n")

	for _, s := range subCommandUsages {
		errorf("\t%s\n", s)
	}

	errorf("Use '%s <subCommand> -h' for subCommand help\n", os.Args[0])
	errorf("\n\tNote that the capitalization of the <subCommand> is ignored.\n")
	os.Exit(1)
}

func allTypesFrequencyRanges() (types []string, freqRanges map[string][]string) {
	freqRanges = codeplug.AllFrequencyRanges()
	types = make([]string, 0, len(freqRanges))

	for typ := range freqRanges {
		types = append(types, typ)
	}

	sort.Strings(types)

	return types, freqRanges
}

func loadCodeplug(fType codeplug.FileType, filename string) (*codeplug.Codeplug, error) {
	cp, err := codeplug.NewCodeplug(fType, filename)
	if err != nil {
		return nil, err
	}

	types, freqs := cp.TypesFrequencyRanges()
	if len(types) == 0 {
		return nil, errors.New("unknown model in codeplug")
	}

	typ := types[0]

	if len(freqs[typ]) == 0 {
		return nil, errors.New("unknown frequency range in codeplug")
	}

	freqRange := freqs[typ][0]

	err = cp.Load(typ, freqRange)
	if err != nil {
		return nil, err
	}

	return cp, nil
}

func progressCallback(aPrefixes []string) func(cur int) error {
	var prefixes []string
	if aPrefixes != nil {
		prefixes = aPrefixes
	}
	prefixIndex := 0
	prefix := prefixes[prefixIndex]
	maxProgress := userdb.MaxProgress
	return func(cur int) error {
		if cur == 0 {
			if prefixIndex != 0 {
				fmt.Println()
			}

			prefix = prefixes[prefixIndex]
			prefixIndex++
		}
		percent := cur * 100 / maxProgress
		fmt.Printf("%s... %3d%%\r", prefix, percent)
		return nil
	}
}

func newCodeplug() error {
	var typ string
	var freq string

	flags := flag.NewFlagSet("newCodeplug", flag.ExitOnError)
	flags.StringVar(&typ, "model", "", "<model name>")
	flags.StringVar(&freq, "freq", "", "<frequency range>")

	flags.Usage = func() {
		errorf("Usage: %s %s -model <modelName> -freq <freqRange> codePlugFilename\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates a new default codeplug for the given radio model.\n\n")
		errorf("\tmodelName must be chosen from the following list,\n")
		errorf("\tand freqRange must be one of its associated values.\n")
		types, freqs := allTypesFrequencyRanges()
		for _, typ := range types {
			errorf("\t\t%s\n", typ)
			for _, freq := range freqs[typ] {
				errorf("\t\t\t%s\n", "\""+freq+"\"")
			}
		}
		os.Exit(1)
	}

	typeFreqs := codeplug.AllFrequencyRanges()

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	if typeFreqs[typ] == nil {
		errorf("bad modelName\n\n")
		flags.Usage()
	}
	freqMap := make(map[string]bool)
	for _, freq := range typeFreqs[typ] {
		freqMap[freq] = true
	}
	if !freqMap[freq] {
		errorf("bad freqRange\n\n")
		flags.Usage()
	}
	filename := args[0]

	cp, err := codeplug.NewCodeplug(codeplug.FileTypeNew, "")
	if err != nil {
		return err
	}

	err = cp.Load(typ, freq)
	if err != nil {
		return err
	}

	return cp.SaveAs(filename)
}

func readCodeplug() error {
	var typ string
	var freq string

	flags := flag.NewFlagSet("readCodeplug", flag.ExitOnError)
	flags.StringVar(&typ, "model", "", "<model name>")
	flags.StringVar(&freq, "freq", "", "<frequency range>")

	flags.Usage = func() {
		errorf("Usage: %s %s -model <modelName> -freq <freqRange> <codePlugFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nReads a codeplug from the radio into <codePlugFilename>.\n\n")
		errorf("\tmodelName must be chosen from the following list,\n")
		errorf("\tand freqRange must be one of its associated values.\n")
		types, freqs := allTypesFrequencyRanges()
		for _, typ := range types {
			errorf("\t\t%s\n", typ)
			for _, freq := range freqs[typ] {
				errorf("\t\t\t%s\n", "\""+freq+"\"")
			}
		}
		os.Exit(1)
	}

	typeFreqs := codeplug.AllFrequencyRanges()

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	if typeFreqs[typ] == nil {
		errorf("bad modelName\n\n")
		flags.Usage()
	}
	freqMap := make(map[string]bool)
	for _, freq := range typeFreqs[typ] {
		freqMap[freq] = true
	}
	if !freqMap[freq] {
		errorf("bad freqRange\n\n")
		flags.Usage()
	}
	filename := args[0]

	cp, err := codeplug.NewCodeplug(codeplug.FileTypeNew, "")
	if err != nil {
		return err
	}

	err = cp.Load(typ, freq)
	if err != nil {
		return err
	}

	prefixes := []string{
		"Preparing to read codeplug",
		"Reading codeplug from radio.",
	}

	err = cp.ReadRadio(progressCallback(prefixes))
	if err != nil {
		return err
	}

	return cp.SaveAs(filename)
}

func writeCodeplug() error {
	flags := flag.NewFlagSet("writeCodeplug", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <codeplugFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nWrites the codeplug in <codeplugFilename> to the radio.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	cp, err := loadCodeplug(codeplug.FileTypeNone, filename)
	if err != nil {
		return err
	}

	prefixes := []string{
		"Preparing to write codeplug to radio",
		"Erasing the radio's codeplug",
		"Writing codeplug to radio",
	}

	return cp.WriteRadio(progressCallback(prefixes))
}

func readSPIFlash() (err error) {
	flags := flag.NewFlagSet("readSPIFlash", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <filename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nReads the contents of the radio's SPI Flash into <filename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	prefixes := []string{
		"Preparing to read flash",
		"Reading flash",
	}

	dfu, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer dfu.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	return dfu.ReadSPIFlash(file)
}

func readMD380Users() (err error) {
	flags := flag.NewFlagSet("readMD380Users", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nReads the user database from the radio to <usersFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}

	filename := args[0]

	prefixes := []string{
		"Preparing to read users",
		fmt.Sprintf("Reading users to %s", filename),
	}

	dfu, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer dfu.Close()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("os.Create: %s", err.Error())
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	return dfu.ReadMD380Users(file)
}

func writeMD380Users() error {
	flags := flag.NewFlagSet("writeMD380Users", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nWrites the user database in <usersFilename> to the radio.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}

	filename := args[0]

	prefixes := []string{
		"Preparing to write users",
		"Erasing flash memory",
		"Writing users",
	}

	dfu, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer dfu.Close()

	db, err := userdb.New(userdb.FromFile(filename), userdb.Abbreviate(false))
	if err != nil {
		return err
	}
	return dfu.WriteMD380Users(db)
}

func writeMD2017Users() error {
	flags := flag.NewFlagSet("writeMD2017Users", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nWrites the user database in <usersFilename> to the radio.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}

	filename := args[0]

	prefixes := []string{
		"Preparing to write users",
		"Erasing flash memory",
		"Writing users",
	}

	df, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer df.Close()

	db, err := userdb.New(userdb.FromFile(filename), userdb.Abbreviate(false))
	if err != nil {
		return err
	}
	return df.WriteUV380Users(db)
}

func writeUV380Users() error {
	flags := flag.NewFlagSet("writeUV380Users", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nWrites the user database in <usersFilename> to the radio.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}

	filename := args[0]

	prefixes := []string{
		"Preparing to write users",
		"Erasing flash memory",
		"Writing users",
	}

	df, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer df.Close()

	db, err := userdb.New(userdb.FromFile(filename), userdb.Abbreviate(false))
	if err != nil {
		return err
	}

	return df.WriteUV380Users(db)
}

func getUsers() error {
	flags := flag.NewFlagSet("getUsers", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nDownloads a curated user database into <usersFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	prefixes := []string{
		"Retrieving Users file",
	}

	db, err := userdb.New(userdb.CuratedUsers(), userdb.Abbreviate(false))
	if err != nil {
		return err
	}

	db.SetProgressCallback(progressCallback(prefixes))
	return db.WriteMD380ToolsFile(filename)
}

func getAbbreviatedUsers() error {
	flags := flag.NewFlagSet("getAbbreviatedUsers", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nDownloads a curated user database into <usersFilename>.\n")
		errorf("The names of many states and countries are abbreviated.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	prefixes := []string{
		"Retrieving Users file",
	}

	db, err := userdb.New(userdb.CuratedUsers(), userdb.Abbreviate(true))
	if err != nil {
		return err
	}

	db.SetProgressCallback(progressCallback(prefixes))
	return db.WriteMD380ToolsFile(filename)
}

func getMergedUsers() error {
	flags := flag.NewFlagSet("getMergedUsers", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nDownloads the user database from multiple websites and merges them\n")
		errorf("into <usersFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	prefixes := []string{
		"Retrieving Users file",
	}

	db, err := userdb.New(userdb.MergeNewUsers(), userdb.Abbreviate(false))
	if err != nil {
		return err
	}

	db.SetProgressCallback(progressCallback(prefixes))
	return db.WriteMD380ToolsFile(filename)
}

func writeMD380Firmware() error {
	flags := flag.NewFlagSet("writeMD380Firmware", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <firmwareFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nWrites the contents of <firmwareFilename> into the MD380 radio.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
	}
	filename := args[0]

	prefixes := []string{
		"Preparing to firmware",
		"Erasing flash memory",
		"Writing firmware",
	}

	dfu, err := dfu.New(progressCallback(prefixes))
	if err != nil {
		return err
	}
	defer dfu.Close()

	file, err := os.Open(filename)
	if err != nil {
		l.Fatalf("writeMD380Firmware: %s", err.Error())
	}

	defer file.Close()

	return dfu.WriteFirmware(file)
}

func textToCodeplug() error {
	flags := flag.NewFlagSet("textToCodeplug", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <textFilename> <codeplugFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates a codeplug file, <codeplugFilename>, from the textual\n")
		errorf("representation in <textFilename>\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	textFilename := args[0]
	codeplugFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeText, textFilename)
	if err != nil {
		return err
	}

	return cp.SaveAs(codeplugFilename)
}

func codeplugToText() error {
	flags := flag.NewFlagSet("codeplugToText", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <codeplugFilename> <textFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates a <textfilename> containing a textual representation of\n")
		errorf("of the codeplug in <codeplugFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	codeplugFilename := args[0]
	textFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeNone, codeplugFilename)
	if err != nil {
		return err
	}

	return cp.ExportText(textFilename)
}

func jsonToCodeplug() error {
	flags := flag.NewFlagSet("jsonToCodeplug", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <jsonFilename> <codeplugFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates a codeplug file, <codeplugFilename>, from the JSON\n")
		errorf("representation in <jsonFilename>\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	jsonFilename := args[0]
	codeplugFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeJSON, jsonFilename)
	if err != nil {
		return err
	}

	return cp.SaveAs(codeplugFilename)
}

func codeplugToJSON() error {
	flags := flag.NewFlagSet("codeplugToJSON", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <codeplugFilename> <jsonFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates <jsonfilename> containing a JSON representation of\n")
		errorf("of the codeplug in <codeplugFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	codeplugFilename := args[0]
	jsonFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeNone, codeplugFilename)
	if err != nil {
		return err
	}

	return cp.ExportJSON(jsonFilename)
}

func xlsxToCodeplug() error {
	flags := flag.NewFlagSet("xlsxToCodeplug", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <xlsxFilename> <codeplugFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates a codeplug file, <codeplugFilename>, from the spreadsheet\n")
		errorf("in <xlsxFilename>\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	xlsxFilename := args[0]
	codeplugFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeXLSX, xlsxFilename)
	if err != nil {
		return err
	}

	return cp.SaveAs(codeplugFilename)
}

func codeplugToXLSX() error {
	flags := flag.NewFlagSet("codeplugToXLSX", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <codeplugFilename> <xlsxFilename>\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nCreates <xlsxfilename> containing a spreadsheet representation of\n")
		errorf("of the codeplug in <codeplugFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}
	codeplugFilename := args[0]
	xlsxFilename := args[1]

	cp, err := loadCodeplug(codeplug.FileTypeNone, codeplugFilename)
	if err != nil {
		return err
	}

	return cp.ExportXLSX(xlsxFilename)
}

func userCountries() error {
	flags := flag.NewFlagSet("userCountries", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename> <countriesFilename>\n", os.Args[0], os.Args[1])
		errorf("  where <usersFilename> is the name of a user file.\n\n")
		flags.PrintDefaults()
		errorf("\nA list of the countries in <usersfilename> will be written to <countriesFilename>.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 2 {
		flags.Usage()
	}

	usersFilename := args[0]
	countriesFilename := args[1]

	db, err := userdb.New(userdb.FromFile(usersFilename), userdb.Abbreviate(false))
	if err != nil {
		return err
	}

	countries, err := db.AllCountries()
	if err != nil {
		return err
	}

	countriesFile, err := os.Create(countriesFilename)
	if err != nil {
		return err
	}

	for _, country := range countries {
		if country == "" {
			country = "<none>"
		}

		fmt.Fprintln(countriesFile, country)
	}

	countriesFile.Close()

	return nil
}

func countryCounts() error {
	flags := flag.NewFlagSet("countryCounts", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <usersFilename>\n", os.Args[0], os.Args[1])
		errorf("  where <usersFilename> is the name of a user file.\n")
		flags.PrintDefaults()
		errorf("\nThe number of users for each country in <usesfilename> will be output.\n")
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()

	usersFilename := args[0]

	db, err := userdb.New(userdb.FromFile(usersFilename), userdb.Abbreviate(false))
	if err != nil {
		return err
	}

	countries, err := db.AllCountries()
	if err != nil {
		return err
	}

	users := db.Users()

	for _, country := range countries {
		count := 0
		for _, user := range users {
			if user.Country == country {
				count++
			}
		}

		if country == "" {
			country = "<none>"
		}

		fmt.Printf("%7d %s\n", count, country)
	}

	fmt.Printf("%7d %s\n", len(users), "Total Users")

	return nil
}

func filterUsers() error {
	flags := flag.NewFlagSet("filterUsers", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s <countriesFile> <inUsersFile> <outUsersFile>\n", os.Args[0], os.Args[1])
		errorf("  where <countriesFile> contains a list of countries, one per line.\n\n")
		errorf("    Blank lines and lines beginning with '#' are ignored.\n")
		errorf("    Only users in the listed countries will be included in the output.\n")
		errorf("  <inUsersFile> is an existing userdb file\n")
		errorf("    If <inUsersFile> is \"\", a curated users file will be downloaded.\n")
		errorf("  <outUsersFile> will be created with users filtered by countries.\n")

		flags.PrintDefaults()
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 3 {
		flags.Usage()
	}
	countriesFilename := args[0]
	inUsersFilename := args[1]
	outUsersFilename := args[2]

	countriesFile, err := os.Open(countriesFilename)
	if err != nil {
		return err
	}

	countries := make([]string, 0)
	scanner := bufio.NewScanner(countriesFile)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.SplitN(line, "#", 2)[0]
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "<none>" {
			line = ""
		}

		countries = append(countries, line)
	}

	db, err := userdb.New(userdb.Abbreviate(false), userdb.FilterByCountries(countries...))
	if err != nil {
		return err
	}
	if inUsersFilename != "" {
		db.SetOptions(userdb.FromFile(inUsersFilename))
	}

	fmt.Println(len(db.Users()), "Users")
	return db.WriteMD380ToolsFile(outUsersFilename)
}

func printVersion() error {
	flags := flag.NewFlagSet("version", flag.ExitOnError)

	flags.Usage = func() {
		errorf("Usage: %s %s\n", os.Args[0], os.Args[1])
		flags.PrintDefaults()
		errorf("\nOutputs the version number of %s.\n", os.Args[0])
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) != 0 {
		flags.Usage()
	}

	fmt.Printf("%s\n", version)
	return nil
}

func main() {
	log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
	log.SetFlags(log.Lshortfile)

	if len(os.Args) < 2 {
		usage()
	}

	subCommandName := strings.ToLower(os.Args[1])

	subCommands := map[string]func() error{
		"newcodeplug":         newCodeplug,
		"readcodeplug":        readCodeplug,
		"writecodeplug":       writeCodeplug,
		"readspiflash":        readSPIFlash,
		"readmd380users":      readMD380Users,
		"writemd380users":     writeMD380Users,
		"writemd2017users":    writeMD2017Users,
		"writeuv380users":     writeUV380Users,
		"getusers":            getUsers,
		"getabbreviatedusers": getAbbreviatedUsers,
		"getmergedusers":      getMergedUsers,
		"writemd380firmware":  writeMD380Firmware,
		"texttocodeplug":      textToCodeplug,
		"codeplugtotext":      codeplugToText,
		"jsontocodeplug":      jsonToCodeplug,
		"codeplugtojson":      codeplugToJSON,
		"xlsxtocodeplug":      xlsxToCodeplug,
		"codeplugtoxlsx":      codeplugToXLSX,
		"usercountries":       userCountries,
		"filterusers":         filterUsers,
		"countrycounts":       countryCounts,
		"version":             printVersion,
	}

	subCommand := subCommands[subCommandName]
	if subCommand == nil {
		usage()
	}

	err := subCommand()
	if err != nil {
		errorf("%s\n", err.Error())
		os.Exit(1)
	}
}
