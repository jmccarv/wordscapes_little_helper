# NAME

Wordscapes Little Helper - Simple web and cli app to find words from a list

# DESCRIPTION

Wordscapes is a mobile game that gives you a group of letters from which you
try to make words that fit in a puzzle. The puzzles contain multiple words
ranging from 3 to 7 letters arranged in a crossword-puzzle style. Sometimes
you will know one or more of the letters in a word, where it crosses another
you've already guessed. Something like this:

    Letters: DIBNRE

      _
     ______
      _
      _

A solution might be:

      B
     BINDER
      N
      D

This application provides two programs:

* extract_wordlist - Extract a list of words from an XML dump of the wiktionary.org wiki
* wlh
    * Perform a oneshot search from the command line
    * Start a web server serving the wlh app

XML dumps of the wiktonary wiki can be found here:

https://dumps.wikimedia.org/enwiktionary/latest/enwiktionary-latest-pages-articles.xml.bz2

# GET THE CODE

Use go get - ignore error: no Go files in ...

    go get bitbucket.org/jmccarv/wordscapes_little_helper

Or clone the git repo:

    git clone https://bitbucket.org/jmccarv/wordscapes_little_helper

# BUILDING

From the root of the project, run:
 ./build.sh


# USE

## Help

Help is available for both commands via --help or -h.

## Create a wordlist file from a wiktionary XML file

    bin/extract_wordlist < wiki.xml > wordlist

## Run a oneshot search from the command line

    bin/wlh -w wordlist find --template .... --letters dibnre

## Web Server

Start the web server:

    bin/wlh -w wordlist serve

Connect to the app:

    http://localhost:8080/

# REPOSITORY

The git repository for this application may be found here:

    https://bitbucket.org/jmccarv/wordscapes_little_helper

# AUTHOR

Jason McCarver <slam@parasite.cc>
