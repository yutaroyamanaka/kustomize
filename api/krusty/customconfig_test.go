// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package krusty_test

import (
	"testing"

	kusttest_test "sigs.k8s.io/kustomize/api/testutils/kusttest"
)

func makeBaseReferencingCustomConfig(th kusttest_test.Harness) {
	th.WriteK("base", `
namePrefix: x-
commonLabels:
  app: myApp
vars:
- name: APRIL_DIET
  objref:
    kind: Giraffe
    name: april
  fieldref:
    fieldpath: spec.diet
- name: KOKO_DIET
  objref:
    kind: Gorilla
    name: koko
  fieldref:
    fieldpath: spec.diet
resources:
- animalPark.yaml
- giraffes.yaml
- gorilla.yaml
configurations:
- config/defaults.yaml
- config/custom.yaml
`)
	th.WriteF("base/giraffes.yaml", `
kind: Giraffe
metadata:
  name: april
spec:
  diet: mimosa
  location: NE
---
kind: Giraffe
metadata:
  name: may
spec:
  diet: acacia
  location: SE
`)
	th.WriteF("base/gorilla.yaml", `
kind: Gorilla
metadata:
  name: koko
spec:
  diet: bambooshoots
  location: SW
`)
	th.WriteF("base/animalPark.yaml", `
apiVersion: foo
kind: AnimalPark
metadata:
  name: sandiego
spec:
  gorillaRef:
    name: koko
  giraffeRef:
    name: april
  food:
  - "$(APRIL_DIET)"
  - "$(KOKO_DIET)"
`)
}

func TestCustomConfig(t *testing.T) {
	th := kusttest_test.MakeHarness(t)
	makeBaseReferencingCustomConfig(th)
	th.WriteLegacyConfigs("base/config/defaults.yaml")
	th.WriteF("base/config/custom.yaml", `
nameReference:
- kind: Gorilla
  fieldSpecs:
  - kind: AnimalPark
    path: spec/gorillaRef/name
- kind: Giraffe
  fieldSpecs:
  - kind: AnimalPark
    path: spec/giraffeRef/name
varReference:
- path: spec/food
  kind: AnimalPark
`)
	m := th.Run("base", th.MakeDefaultOptions())
	th.AssertActualEqualsExpected(m, `
apiVersion: foo
kind: AnimalPark
metadata:
  labels:
    app: myApp
  name: x-sandiego
spec:
  food:
  - mimosa
  - bambooshoots
  giraffeRef:
    name: x-april
  gorillaRef:
    name: x-koko
---
kind: Giraffe
metadata:
  labels:
    app: myApp
  name: x-april
spec:
  diet: mimosa
  location: NE
---
kind: Giraffe
metadata:
  labels:
    app: myApp
  name: x-may
spec:
  diet: acacia
  location: SE
---
kind: Gorilla
metadata:
  labels:
    app: myApp
  name: x-koko
spec:
  diet: bambooshoots
  location: SW
`)
}

func TestCustomConfigWithDefaultOverspecification(t *testing.T) {
	th := kusttest_test.MakeHarness(t)
	makeBaseReferencingCustomConfig(th)
	th.WriteLegacyConfigs("base/config/defaults.yaml")
	// Specifying namePrefix here conflicts with (is the same as)
	// the defaults written above.  This is intentional in the
	// test to assure duplicate config doesn't cause problems.
	th.WriteF("base/config/custom.yaml", `
namePrefix:
- path: metadata/name
nameReference:
- kind: Gorilla
  fieldSpecs:
  - kind: AnimalPark
    path: spec/gorillaRef/name
- kind: Giraffe
  fieldSpecs:
  - kind: AnimalPark
    path: spec/giraffeRef/name
varReference:
- path: spec/food
  kind: AnimalPark
`)
	m := th.Run("base", th.MakeDefaultOptions())
	th.AssertActualEqualsExpected(m, `
apiVersion: foo
kind: AnimalPark
metadata:
  labels:
    app: myApp
  name: x-sandiego
spec:
  food:
  - mimosa
  - bambooshoots
  giraffeRef:
    name: x-april
  gorillaRef:
    name: x-koko
---
kind: Giraffe
metadata:
  labels:
    app: myApp
  name: x-april
spec:
  diet: mimosa
  location: NE
---
kind: Giraffe
metadata:
  labels:
    app: myApp
  name: x-may
spec:
  diet: acacia
  location: SE
---
kind: Gorilla
metadata:
  labels:
    app: myApp
  name: x-koko
spec:
  diet: bambooshoots
  location: SW
`)
}

func TestFixedBug605_BaseCustomizationAvailableInOverlay(t *testing.T) {
	th := kusttest_test.MakeHarness(t)
	makeBaseReferencingCustomConfig(th)
	th.WriteLegacyConfigs("base/config/defaults.yaml")
	th.WriteF("base/config/custom.yaml", `
nameReference:
- kind: Gorilla
  fieldSpecs:
  - apiVersion: foo
    kind: AnimalPark
    path: spec/gorillaRef/name
- kind: Giraffe
  fieldSpecs:
  - apiVersion: foo
    kind: AnimalPark
    path: spec/giraffeRef/name
varReference:
- path: spec/food
  apiVersion: foo
  kind: AnimalPark
`)
	th.WriteK("overlay", `
namePrefix: o-
commonLabels:
  movie: planetOfTheApes
patchesStrategicMerge:
- animalPark.yaml
resources:
- ../base
- ursus.yaml
`)
	th.WriteF("overlay/ursus.yaml", `
kind: Gorilla
metadata:
  name: ursus
spec:
  diet: heston
  location: Arizona
`)
	// The following replaces the gorillaRef in the AnimalPark.
	th.WriteF("overlay/animalPark.yaml", `
apiVersion: foo
kind: AnimalPark
metadata:
  name: sandiego
spec:
  gorillaRef:
    name: ursus
`)
	m := th.Run("overlay", th.MakeDefaultOptions())
	th.AssertActualEqualsExpected(m, `
apiVersion: foo
kind: AnimalPark
metadata:
  labels:
    app: myApp
    movie: planetOfTheApes
  name: o-x-sandiego
spec:
  food:
  - mimosa
  - bambooshoots
  giraffeRef:
    name: o-x-april
  gorillaRef:
    name: o-ursus
---
kind: Giraffe
metadata:
  labels:
    app: myApp
    movie: planetOfTheApes
  name: o-x-april
spec:
  diet: mimosa
  location: NE
---
kind: Giraffe
metadata:
  labels:
    app: myApp
    movie: planetOfTheApes
  name: o-x-may
spec:
  diet: acacia
  location: SE
---
kind: Gorilla
metadata:
  labels:
    app: myApp
    movie: planetOfTheApes
  name: o-x-koko
spec:
  diet: bambooshoots
  location: SW
---
kind: Gorilla
metadata:
  labels:
    movie: planetOfTheApes
  name: o-ursus
spec:
  diet: heston
  location: Arizona
`)
}

func TestCustomConfigLabelsMerge(t *testing.T) {
	th := kusttest_test.MakeHarness(t)
	th.WriteK("base", `
commonLabels:
  app: myApp
labels:
- pairs:
    giraffe: giraffe
resources:
- animalPark.yaml
configurations:
- config/defaults.yaml
- config/custom.yaml
`)
	th.WriteF("base/animalPark.yaml", `
apiVersion: foo
kind: AnimalPark
metadata:
  name: sandiego
spec:
  giraffeRef:
    name: april
`)
	th.WriteLegacyConfigs("base/config/defaults.yaml")
	th.WriteF("base/config/custom.yaml", `
commonLabels:
- kind: AnimalPark
  path: spec/giraffeRef/metadata/labels
  create: true
labels:
- kind: AnimalPark
  path: spec/giraffeRef/metadata/labels
  create: true
`)
	m := th.Run("base", th.MakeDefaultOptions())
	th.AssertActualEqualsExpected(m, `
apiVersion: foo
kind: AnimalPark
metadata:
  labels:
    app: myApp
    giraffe: giraffe
  name: sandiego
spec:
  giraffeRef:
    metadata:
      labels:
        app: myApp
        giraffe: giraffe
    name: april
`)
}
