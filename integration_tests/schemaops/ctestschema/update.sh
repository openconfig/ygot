#!/bin/bash

# Hack to ensure that if we are running on OS X with a homebrew installed
# GNU sed then we can still run sed.
runsed() {
  if hash gsed 2>/dev/null; then
    gsed "$@"
  else
    sed "$@"
  fi
}

go run ../../../generator/generator.go -path="." -output_file=ctestschema.go \
  -package_name=ctestschema -generate_fakeroot -fakeroot_name=device \
  -compress_paths \
  -shorten_enum_leaf_names \
  -typedef_enum_with_defmod \
  -ignore_shadow_schema_paths \
  -enum_suffix_for_simple_union_enums \
  -generate_rename \
  -generate_append \
  -generate_getters \
  -generate_leaf_getters \
  -generate_simple_unions \
  -generate_populate_defaults \
  -annotations \
  ../yang/ctestschema.yang ../yang/ctestschema-rootmod.yang
gofmt -w -s ctestschema.go
