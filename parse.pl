#!/usr/bin/env perl

# Q&D script to parse the pages-articles.xml file
# from wiktionary to get a list of english words.
# Not as straightforward as it should be. Have to
# check the category data in the text to make sure
# it's an english word. This may not even be complete/right

use strict;
use warnings;
use Data::Dumper;
$Data::Dumper::Sortkeys = 1;

my %whitelist = ( initialism => 1, surname => 1 );
my %list;
my %plural;
my $word;
my $good;
my $plural_of;
while (<>) {
    if (/<title>([a-zA-Z]*)<\/title>/) {
        if ($word && ($good || $whitelist{$word})) {
            $list{$word} = 1;
            $plural{$word} = $plural_of if $plural_of;
        }

        $word = lc($1);
        $plural_of = '';

        if ($list{$word}) {
            $word = '';
            next;
        }

        $good = 0;
        next;
    }

    next unless $word;

    if (/initialism|surname\|lang=en/i) {
        $word = '';
        next;
    }

    if (/plural of\|(\w+)\|lang=en/) {
        $plural_of = lc($1);
    }
    
    if (/==English==|Category:(en[^a-z]|English)/) {
        $good = 1;
    }
}
if ($word && ($good || $whitelist{$word})) {
    $list{$word} = 1;
    $plural{$word} = $plural_of if $plural_of;
}

# print STDERR Dumper(\%plural);

# Now remove any plurals whose base word we rejected
# This can happen for example with an initialism where the base
# word shows to be an initialism but the plural does not
# See aon / aons for example
for (keys %plural) {
    delete $list{$_} unless $list{$plural{$_}}
}

#print "plurals:\n";
#print Dumper(\%plural);

#print "\nword list:\n";
print "$_\n" for sort keys %list
