#!/bin/bash

#
# Takes the compressed pages-articles.xml file
#  https://dumps.wikimedia.org/enwiktionary/latest/enwiktionary-latest-all-titles-in-ns0.gz 
#
# Outputs a list of words 3-7 characters long sorted by word length
#

time pv *pages-articles.xml.gz | pigz -d -c | ./parse.pl | \
#./parse.pl t.fil | \
grep '^[A-Za-z]*$' | tr '[A-Z]' '[a-z]' | \
sort | uniq | \
awk '{ print length, $0 }' | grep '^ *[3-7] ' | \
sort -n -s | cut -d ' ' -f 2-
